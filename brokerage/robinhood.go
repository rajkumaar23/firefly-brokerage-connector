package brokerage

import (
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
)

type Robinhood struct {
	RestyClient *resty.Client
	token       string
}

type LoginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	GrantType   string `json:"grant_type"`
	ClientID    string `json:"client_id"`
	ExpiresIn   uint64 `json:"expires_in"`
	Scope       string `json:"scope"`
	DeviceToken string `json:"device_token"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type AccountsResponse struct {
	Results []struct {
		AccountNumber uint64 `json:"account_number,string"`
	}
}

type PortfolioResponse struct {
	ExtendedHoursEquity float64 `json:"extended_hours_equity,string"`
}

type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
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
		SetBody(LoginRequest{
			Username:    os.Getenv("RH_USERNAME"),
			Password:    os.Getenv("RH_PASSWORD"),
			GrantType:   "password",
			ClientID:    os.Getenv("RH_CLIENT_ID"),
			ExpiresIn:   300,
			Scope:       "internal",
			DeviceToken: os.Getenv("RH_DEVICE_TOKEN"),
		}).
		SetResult(LoginResponse{}).
		SetError(ErrorResponse{}).
		Post("/oauth2/token/")

	if err != nil {
		return err
	}

	if resp.Error() != nil {
		return fmt.Errorf("%+v", resp.Error().(*ErrorResponse))
	}

	r.RestyClient.SetAuthToken(resp.Result().(*LoginResponse).AccessToken)
	return nil
}

func (r *Robinhood) GetBalance() (float64, error) {
	resp, err := r.RestyClient.R().
		SetResult(AccountsResponse{}).
		SetError(ErrorResponse{}).
		Get("/accounts/?default_to_all_accounts=true")

	if err != nil {
		return 0, err
	}

	if resp.Error() != nil {
		return 0, fmt.Errorf("%+v", resp.Error().(*ErrorResponse))
	}

	portfolioURL := fmt.Sprintf("/portfolios/%d", resp.Result().(*AccountsResponse).Results[0].AccountNumber)
	resp, err = r.RestyClient.R().
		SetResult(PortfolioResponse{}).
		SetError(ErrorResponse{}).
		Get(portfolioURL)

	if err != nil {
		return 0, err
	}

	if resp.Error() != nil {
		return 0, fmt.Errorf("%+v", resp.Error().(*ErrorResponse))
	}

	return resp.Result().(*PortfolioResponse).ExtendedHoursEquity, nil
}
