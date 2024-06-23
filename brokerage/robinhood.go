package brokerage

import (
	"fmt"
	"os"
	"strconv"

	"github.com/go-resty/resty/v2"
)

type Robinhood struct {
	RestyClient *resty.Client
}

type RobinhoodLoginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	GrantType   string `json:"grant_type"`
	ClientID    string `json:"client_id"`
	ExpiresIn   uint64 `json:"expires_in"`
	Scope       string `json:"scope"`
	DeviceToken string `json:"device_token"`
}

type RobinhoodLoginResponse struct {
	AccessToken string `json:"access_token"`
}

type RobinhoodAccountsResponse struct {
	Results []struct {
		AccountNumber uint64 `json:"account_number,string"`
	}
}

type RobinhoodPortfolioResponse struct {
	ExtendedHoursEquity float64 `json:"extended_hours_equity,string"`
}

type RobinhoodErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

func (r *Robinhood) Name() string {
	return "robinhood"
}

func (r *Robinhood) Currency() string {
	return "dollars"
}

func (r *Robinhood) FireflyAccountID() (uint8, error) {
	id, err := strconv.Atoi(os.Getenv("ROBINHOOD_FIREFLY_ID"))
	if err != nil {
		return 0, err
	}

	return uint8(id), nil
}

func (r *Robinhood) Prepare() {
	r.RestyClient = resty.New()
	r.RestyClient.SetBaseURL("https://api.robinhood.com")
	r.RestyClient.SetHeaders(map[string]string{
		"Accept":                  "*/*",
		"Accept-Encoding":         "gzip, deflate",
		"Accept-Language":         "en-GB,en;q=0.9",
		"X-Robinhood-API-Version": "1.431.4",
		"Connection":              "keep-alive",
		"User-Agent":              "Robinhood/823 (iPhone; iOS 7.1.2; Scale/2.00)",
	})

}

func (r *Robinhood) Login() error {
	resp, err := r.RestyClient.R().
		SetBody(RobinhoodLoginRequest{
			Username:    os.Getenv("RH_USERNAME"),
			Password:    os.Getenv("RH_PASSWORD"),
			GrantType:   "password",
			ClientID:    os.Getenv("RH_CLIENT_ID"),
			ExpiresIn:   300,
			Scope:       "internal",
			DeviceToken: os.Getenv("RH_DEVICE_TOKEN"),
		}).
		SetResult(RobinhoodLoginResponse{}).
		SetError(RobinhoodErrorResponse{}).
		Post("/oauth2/token/")

	if err != nil {
		return fmt.Errorf("error making login request: %v", err)
	}

	if resp.Error() != nil {
		return fmt.Errorf("error in login response: %+v", resp.Error().(*RobinhoodErrorResponse))
	}

	fmt.Println("robinhood: login success ✅")

	r.RestyClient.SetAuthToken(resp.Result().(*RobinhoodLoginResponse).AccessToken)
	return nil
}

func (r *Robinhood) GetBalance() (float64, error) {
	resp, err := r.RestyClient.R().
		SetResult(RobinhoodAccountsResponse{}).
		SetError(RobinhoodErrorResponse{}).
		Get("/accounts/?default_to_all_accounts=true")

	if err != nil {
		return 0, fmt.Errorf("error making list-accounts request: %v", err)
	}

	if resp.Error() != nil {
		return 0, fmt.Errorf("error in list-accounts response: %+v", resp.Error().(*RobinhoodErrorResponse))
	}

	fmt.Println("robinhood: accounts list fetched ✅")

	portfolioURL := fmt.Sprintf("/portfolios/%d", resp.Result().(*RobinhoodAccountsResponse).Results[0].AccountNumber)
	resp, err = r.RestyClient.R().
		SetResult(RobinhoodPortfolioResponse{}).
		SetError(RobinhoodErrorResponse{}).
		Get(portfolioURL)

	if err != nil {
		return 0, fmt.Errorf("error making portfolio request: %v", err)
	}

	if resp.Error() != nil {
		return 0, fmt.Errorf("error in portfolio response: %+v", resp.Error().(*RobinhoodErrorResponse))
	}

	fmt.Println("robinhood: portfolio fetched ✅")

	return resp.Result().(*RobinhoodPortfolioResponse).ExtendedHoursEquity, nil
}
