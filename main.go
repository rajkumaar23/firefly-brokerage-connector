package main

import (
	"fmt"

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
	brokerages := []brokerage.Brokerage{
		&brokerage.Robinhood{},
	}

	for _, broker := range brokerages {
		broker.Prepare()

		err := broker.Login()
		if err != nil {
			fmt.Print(err)
		}

		portfolio, err := broker.GetBalance()
		if err != nil {
			fmt.Print(err)
		}

		fmt.Printf("Current equity balance: $%.2f\n", portfolio)
	}

}
