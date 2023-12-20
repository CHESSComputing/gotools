package cmd

import (
	"fmt"
	"os"
)

func checkError(err error) {
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
}

func exit(msg string, err error) {
	fmt.Println("ERROR", msg, err)
	os.Exit(1)
}
