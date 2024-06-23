package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	brokerage "github.com/rajkumaar23/firefly-brokerage-connector/brokerage"
	firefly "github.com/rajkumaar23/firefly-brokerage-connector/firefly"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("error loading .env file: %v\n", err)
		fmt.Println("ensure all the environment variables are set!")
	}
}

func main() {
	ff := &firefly.Firefly{}
	ff.Prepare()

	brokerages := []brokerage.Brokerage{
		&brokerage.Robinhood{},
		&brokerage.Zerodha{},
	}

	exitCode := 0
	for _, broker := range brokerages {
		fireflyAccountID, err := broker.FireflyAccountID()
		if err != nil {
			exitCode = 1
			fmt.Printf("%s: error getting firefly account ID: %v", broker.Name(), err)
			continue
		}

		fireflyBalance, err := ff.GetBalance(fireflyAccountID)
		if err != nil {
			exitCode = 1
			fmt.Printf("%s: error getting firefly balance: %v", broker.Name(), err)
			continue
		}

		broker.Prepare()

		err = broker.Login()
		if err != nil {
			exitCode = 1
			fmt.Printf("%s: error in login: %v", broker.Name(), err)
			continue
		}

		balance, err := broker.GetBalance()
		if err != nil {
			fmt.Printf("%s: error fetching balance: %v", broker.Name(), err)
			exitCode = 1
			continue
		}

		fmt.Printf("%s: current balance = %.2f %s 💰\n", broker.Name(), balance, broker.Currency())
		fmt.Printf("%s: firefly balance = %.2f %s 💰\n", broker.Name(), fireflyBalance, broker.Currency())

		err = nil
		if balance < fireflyBalance {
			err = ff.PostTransaction(fireflyAccountID, firefly.Withdrawal, fireflyBalance-balance)
			if err == nil {
				fmt.Printf("%s: balance updated 📉\n", broker.Name())
			}
		} else if balance > fireflyBalance {
			err = ff.PostTransaction(fireflyAccountID, firefly.Deposit, balance-fireflyBalance)
			if err == nil {
				fmt.Printf("%s: balance updated 📈\n", broker.Name())
			}
		}

		if err != nil {
			fmt.Printf("%s: error updating balance: %v\n", broker.Name(), err)
			exitCode = 1
		}

		fmt.Printf("--------------------------------------------------------------------------------\n")
	}

	os.Exit(exitCode)
}
