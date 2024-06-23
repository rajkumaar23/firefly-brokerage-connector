package brokerage

type Brokerage interface {
	Login() error
	Name() string
	Currency() string
	FireflyAccountID() (uint8, error)
	Prepare()
	GetBalance() (float64, error)
}
