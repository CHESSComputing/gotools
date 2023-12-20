package cmd

import (
	"fmt"
	"os"
)

var doc = "Complete documentation at http://www.lepp.cornell.edu/CHESSComputing"

// helper function to exit with message and error
func exit(msg string, err error) {
	fmt.Println("ERROR", msg, err)
	os.Exit(1)
}
