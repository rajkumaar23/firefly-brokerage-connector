package brokerage

import (
	"os"
	"strconv"
)

type Fidelity struct{}

func (f *Fidelity) Prepare() {
	// nothing to prepare
}

func (f *Fidelity) Login() error {
	// nowhere to login
	return nil
}

func (f *Fidelity) Name() string {
	return "fidelity"
}

func (f *Fidelity) Currency() string {
	return "dollars"
}

func (f *Fidelity) FireflyAccountID() (uint8, error) {
	id, err := strconv.Atoi(os.Getenv("FIDELITY_FIREFLY_ID"))
	if err != nil {
		return 0, err
	}

	return uint8(id), nil
}

func (f *Fidelity) GetBalance() (float64, error) {
	envBalance := os.Getenv("FIDELITY_BALANCE")
	balance, err := strconv.ParseFloat(envBalance, 32)
	if err != nil {
		return 0, err
	}
	return balance, nil
}
