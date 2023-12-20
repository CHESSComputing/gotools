package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	authz "github.com/CHESSComputing/golib/authz"
	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

// helper function to provide usage of meta option
func authUsage() {
	fmt.Println("client auth kerberos")
	fmt.Println("client auth token")
}

func requestToken(fname string) (string, error) {
	var token string
	user, ticket := getKerberosTicket(fname)
	rec := authz.Kerberos{
		User:   user,
		Ticket: ticket,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return token, err
	}
	httpMgr := services.NewHttpRequest("read", 1)
	rurl := fmt.Sprintf("%s/oauth/authorize", _srvConfig.Services.AuthzURL)
	ctype := "applicatin/json"
	buf := bytes.NewBuffer(data)
	resp, err := httpMgr.Post(rurl, ctype, buf)
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}
	var response map[string]any
	err = json.Unmarshal(data, &response)
	if err != nil {
		return token, err
	}
	if token, ok := response["access_token"]; ok {
		return fmt.Sprintf("%v", token), nil
	}
	msg := fmt.Sprintf("Unable to obtain valid token, Auth service response %+v", response)
	return token, errors.New(msg)
}

func authCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "client auth command",
		Long:  "client auth command\n" + doc,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				authUsage()
			} else if args[0] == "token" {
				fname := args[1]
				token, err := requestToken(fname)
				if err != nil {
					exit("unable to get valid token", err)
				}
				fmt.Println("token", token)
				fmt.Println("Please put it into CHESS_TOKEN env variable to re-use in other comands")
			} else {
				fmt.Println("ERROR")
			}
		},
	}
	return cmd
}
