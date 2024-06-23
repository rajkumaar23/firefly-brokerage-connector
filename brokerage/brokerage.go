package brokerage

type Brokerage interface {
	Login() error
	Name() string
	Currency() string
	Prepare()
	GetBalance() (float64, error)
}
