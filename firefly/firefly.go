package firefly

import (
	"fmt"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

type Firefly struct {
	RestyClient *resty.Client
}

type AccountResponse struct {
	Data struct {
		Attributes struct {
			Balance float64 `json:"current_balance,string"`
		} `json:"attributes"`
	} `json:"data"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type TransactionType string

const (
	Withdrawal TransactionType = "withdrawal"
	Deposit    TransactionType = "deposit"
)

type Transaction struct {
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
	SourceID      uint8   `json:"source_id"`
	DestinationID uint8   `json:"destination_id"`
	Date          string  `json:"date"`
}

type TransactionRequest struct {
	Transactions []Transaction `json:"transactions"`
}

func (f *Firefly) Prepare() {
	f.RestyClient = resty.New()
	f.RestyClient.SetBaseURL(os.Getenv("FIREFLY_HOST"))
	f.RestyClient.SetAuthToken(os.Getenv("FIREFLY_TOKEN"))
}

func (f *Firefly) GetBalance(accountID uint8) (float64, error) {
	resp, err := f.RestyClient.R().
		SetResult(AccountResponse{}).
		SetError(ErrorResponse{}).
		Get(fmt.Sprintf("/api/v1/accounts/%d", accountID))
	if err != nil {
		return 0, fmt.Errorf("error fetching balance: %v", err)
	}
	if resp.Error() != nil {
		return 0, fmt.Errorf("error in fetch balance response: %+v", resp.Error().(*ErrorResponse))
	}

	accountResponse := resp.Result().(*AccountResponse)
	return accountResponse.Data.Attributes.Balance, nil
}

func (f *Firefly) PostTransaction(accountID uint8, transferType TransactionType, amount float64) error {
	var transaction Transaction
	if transferType == Withdrawal {
		transaction = Transaction{
			Type:        string(Withdrawal),
			Amount:      amount,
			Description: "Loss",
			SourceID:    accountID,
			Date:        time.Now().Format(time.RFC3339),
		}
	} else {
		transaction = Transaction{
			Type:          string(Deposit),
			Amount:        amount,
			Description:   "Profit",
			DestinationID: accountID,
			Date:          time.Now().Format(time.RFC3339),
		}
	}
	resp, err := f.RestyClient.R().
		SetBody(TransactionRequest{
			Transactions: []Transaction{
				transaction,
			},
		}).
		SetError(ErrorResponse{}).
		Post("/api/v1/transactions")
	if err != nil {
		return fmt.Errorf("error posting transaction: %v", err)
	}
	if resp.Error() != nil {
		return fmt.Errorf("error in post transaction response: (%d) %+v", resp.StatusCode(), resp.Error().(*ErrorResponse))
	}
	return nil
}
