package cmd

import (
	"fmt"
	"os"

	services "github.com/CHESSComputing/golib/services"
	"github.com/CHESSComputing/golib/utils"
)

var doc = "Complete documentation at http://www.lepp.cornell.edu/CHESSComputing"

var _httpReadRequest, _httpWriteRequest *services.HttpRequest

// helper function to exit with message and error
func exit(msg string, err error) {
	fmt.Println("ERROR", msg, err)
	os.Exit(1)
}

// helper function to obtain read access token
func accessToken() (string, error) {
	if _httpReadRequest.Token == "" {
		token := utils.ReadToken(os.Getenv("CHESS_TOKEN"))
		if token == "" {
			exit("Please obtain read access token and put it into CHESS_TOKEN env or file", nil)
		}
		_httpReadRequest.Token = token
	}
	return _httpReadRequest.Token, nil
}

// helper function to obtain write access token
func writeToken() (string, error) {
	if _httpWriteRequest.Token == "" {
		token := utils.WriteToken(os.Getenv("CHESS_WRITE_TOKEN"))
		if token == "" {
			exit("Please obtain write access token and put it into CHESS_WRITE_TOKEN env or file", nil)
		}
		_httpWriteRequest.Token = token
	}
	return _httpWriteRequest.Token, nil
}
