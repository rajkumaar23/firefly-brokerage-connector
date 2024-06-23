package brokerage

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pquerna/otp/totp"
)

type Zerodha struct {
	ConsoleRestyClient *resty.Client
	KiteRestyClient    *resty.Client
}

type ZerodhaLoginResponse struct {
	Data struct {
		RequestID string `json:"request_id"`
	} `json:"data"`
}

type ZerodhaDashboardResponse struct {
	Data struct {
		Result struct {
			EquityHoldings     float64 `json:"eq_holdings_value"`
			MutualFundHoldings float64 `json:"mf_holdings_value"`
		} `json:"result"`
	} `json:"data"`
}

type ZerodhaErrorResponse struct {
	Message string `json:"message"`
}

func (z *Zerodha) Name() string {
	return "zerodha"
}

func (z *Zerodha) Currency() string {
	return "rupees"
}

func (z *Zerodha) FireflyAccountID() (uint8, error) {
	id, err := strconv.Atoi(os.Getenv("ZERODHA_FIREFLY_ID"))
	if err != nil {
		return 0, err
	}

	return uint8(id), nil
}

func (z *Zerodha) Prepare() {
	z.KiteRestyClient = resty.New()
	z.KiteRestyClient.SetBaseURL("https://kite.zerodha.com")
	z.KiteRestyClient.SetRedirectPolicy(resty.NoRedirectPolicy())

	z.ConsoleRestyClient = resty.New()
	z.ConsoleRestyClient.SetBaseURL("https://console.zerodha.com")
	z.ConsoleRestyClient.SetRedirectPolicy(resty.NoRedirectPolicy())
}

func (z *Zerodha) Login() error {
	err := z.establishConsoleSession()
	if err != nil {
		return fmt.Errorf("error establishing console session: %v", err)
	}
	fmt.Println("zerodha: console session established ✅")

	err = z.establishKiteSession()
	if err != nil {
		return fmt.Errorf("error establishing kite session: %v", err)
	}
	fmt.Println("zerodha: kite session established ✅")

	requestID, err := z.inititateLogin()
	if err != nil {
		return fmt.Errorf("error initiating kite login: %v", err)
	}
	fmt.Println("zerodha: kite login success ✅")

	err = z.initiateMFA(requestID)
	if err != nil {
		return fmt.Errorf("error initiating kite MFA: %v", err)
	}
	fmt.Println("zerodha: kite mfa success ✅")

	location, err := z.finishKiteSession()
	if err != nil {
		return fmt.Errorf("error finishing kite session: %v", err)
	}
	fmt.Println("zerodha: kite session completed ✅")

	err = z.finishConsoleAuthorization(location)
	if err != nil {
		return fmt.Errorf("error finishing console authorization: %v", err)
	}
	fmt.Println("zerodha: console authorization completed ✅")

	return nil
}

func (z *Zerodha) GetBalance() (float64, error) {
	resp, err := z.ConsoleRestyClient.R().SetResult(ZerodhaDashboardResponse{}).SetError(ZerodhaErrorResponse{}).Get("/api/dashboard")
	if err != nil {
		return 0, err
	}
	if resp.Error() != nil {
		return 0, fmt.Errorf("%+v", resp.Error().(*ZerodhaErrorResponse))
	}

	fmt.Println("zerodha: balance fetched ✅")

	dashboardAPIResp := resp.Result().(*ZerodhaDashboardResponse)
	return dashboardAPIResp.Data.Result.EquityHoldings + dashboardAPIResp.Data.Result.MutualFundHoldings, nil
}

// ****************** Internal helper methods ******************

func (z *Zerodha) establishConsoleSession() error {
	resp, err := z.ConsoleRestyClient.R().Get("/")
	if err != nil {
		return err
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			z.ConsoleRestyClient.SetCookie(cookie)
			return nil
		}
	}

	return fmt.Errorf("session cookie not found: %+v", resp.Cookies())
}

func (z *Zerodha) establishKiteSession() error {
	resp, err := z.KiteRestyClient.R().Get("/connect/login?api_key=console&v=3")
	if err != nil && !strings.HasSuffix(err.Error(), resty.ErrAutoRedirectDisabled.Error()) {
		return err
	}
	if resp.StatusCode() != 302 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "kf_session" {
			z.KiteRestyClient.SetCookie(cookie)
			goto cookieFound
		}
	}

	return fmt.Errorf("kf_session cookie not found: %+v", resp.Cookies())

cookieFound:
	redirectURL, err := url.Parse(resp.Header().Get("location"))
	if err != nil {
		return fmt.Errorf("could not parse redirect URL: %v", err)
	}

	z.KiteRestyClient.SetQueryParam("sess_id", redirectURL.Query().Get("sess_id"))
	return nil
}

func (z *Zerodha) inititateLogin() (string, error) {
	resp, err := z.KiteRestyClient.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{"user_id": os.Getenv("ZERODHA_USER_ID"), "password": os.Getenv("ZERODHA_PASSWORD")}).
		SetResult(ZerodhaLoginResponse{}).
		SetError(ZerodhaErrorResponse{}).
		Post("/api/login")

	if err != nil {
		return "", err
	}

	if resp.Error() != nil {
		return "", fmt.Errorf("%+v", resp.Error().(*ZerodhaErrorResponse))
	}

	loginRequestID := resp.Result().(*ZerodhaLoginResponse).Data.RequestID
	if loginRequestID == "" {
		return "", fmt.Errorf("request_id not found: %s", resp.Body())
	}

	return loginRequestID, nil
}

func (z *Zerodha) initiateMFA(loginRequestID string) error {
	otp, err := totp.GenerateCode(os.Getenv("ZERODHA_TOTP_SECRET"), time.Now())
	if err != nil {
		return fmt.Errorf("totp generation failed: %v", err)
	}

	resp, err := z.KiteRestyClient.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"user_id":      os.Getenv("ZERODHA_USER_ID"),
			"request_id":   loginRequestID,
			"twofa_type":   "totp",
			"twofa_value":  otp,
			"skip_session": "true",
		}).
		SetError(ZerodhaErrorResponse{}).
		Post("/api/twofa")

	if err != nil {
		return err
	}

	if resp.Error() != nil {
		return fmt.Errorf("%+v", resp.Error().(*ZerodhaErrorResponse))
	}

	cookiesToSearch := map[string]bool{
		"public_token": false,
		"enctoken":     false,
		"user_id":      false,
	}

	for _, cookie := range resp.Cookies() {
		if _, ok := cookiesToSearch[cookie.Name]; ok {
			cookiesToSearch[cookie.Name] = true
			switch cookie.Name {
			case "public_token":
				z.ConsoleRestyClient.SetCookie(cookie)
				z.ConsoleRestyClient.SetHeader("x-csrftoken", cookie.Value)
				fallthrough
			case "enctoken":
				fallthrough
			case "user_id":
				z.KiteRestyClient.SetCookie(cookie)
			}
		}

	}

	for cookie, found := range cookiesToSearch {
		if !found {
			return fmt.Errorf("cookie not found in MFA response: %s", cookie)
		}
	}

	return nil
}

func (z *Zerodha) finishKiteSession() (string, error) {
	resp, err := z.KiteRestyClient.R().Get("/connect/finish?api_key=console")
	if err != nil && !strings.HasSuffix(err.Error(), resty.ErrAutoRedirectDisabled.Error()) {
		return "", err
	}
	if resp.StatusCode() != 302 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	location := resp.Header().Get("location")
	if location == "" {
		return "", fmt.Errorf("location header not found")
	}

	return location, nil
}

func (z *Zerodha) finishConsoleAuthorization(location string) error {
	resp, err := z.ConsoleRestyClient.R().Get(location)
	if err != nil && !strings.HasSuffix(err.Error(), resty.ErrAutoRedirectDisabled.Error()) {
		return err
	}
	if resp.StatusCode() != 302 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}
