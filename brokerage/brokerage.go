package brokerage

type Brokerage interface {
	Login() error
	Prepare()
	GetBalance() (float64, error)
}
