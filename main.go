package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	brokerage "github.com/rajkumaar23/firefly-brokerage-connector/brokerage"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("error loading .env file: %v\n", err)
		fmt.Println("ensure all the environment variables are set!")
	}
}

func main() {
	exitCode := 0
	brokerages := []brokerage.Brokerage{
		&brokerage.Robinhood{},
		&brokerage.Zerodha{},
	}

	for _, broker := range brokerages {
		broker.Prepare()

		err := broker.Login()
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
	}

	os.Exit(exitCode)
}
