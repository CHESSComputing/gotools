package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	authz "github.com/CHESSComputing/golib/authz"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// helper function to provide usage of meta option
func authUsage() {
	fmt.Println("client auth kerberos")
	fmt.Println("client auth token")
}

func requestToken(scope, fname string) (string, error) {
	var token string
	user, ticket := getKerberosTicket(fname)
	rec := authz.Kerberos{
		User:   user,
		Scope:  scope,
		Ticket: ticket,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return token, err
	}
	httpMgr := services.NewHttpRequest(scope, 1)
	rurl := fmt.Sprintf("%s/oauth/authorize", _srvConfig.Services.AuthzURL)
	ctype := "applicatin/json"
	buf := bytes.NewBuffer(data)
	//     log.Println("### call", rurl)
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
	if tokenScope, ok := response["scope"]; ok {
		if tokenScope != scope {
			return "", errors.New("wrong token scope")
		}
	}
	msg := fmt.Sprintf("Unable to obtain valid token, Auth service response %+v", response)
	return token, errors.New(msg)
}

func inspectAllTokens() {
	inspectToken("CHESS_TOKEN")
	inspectToken("CHESS_WRITE_TOKEN")
}

func inspectToken(tkn string) {
	token := utils.ReadToken(os.Getenv(tkn))
	fmt.Println(tkn, token)
	claims := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
	rclaims := claims.RegisteredClaims
	fmt.Println("Issuer       : ", rclaims.Issuer)
	fmt.Println("Subject      : ", rclaims.Subject)
	fmt.Println("Audience     : ", rclaims.Audience)
	fmt.Println("ExpiresAt    : ", rclaims.ExpiresAt)
	fmt.Printf("Custom Claims: %+v", claims.CustomClaims)
}

func authCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "client authentication/authorization commands",
		Long:  "client authentication/authorization commands\n" + doc,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				authUsage()
			} else if args[0] == "token" {
				attr := args[1]
				var token, tokenKind string
				var err error
				if attr == "view" || attr == "inspect" {
					inspectAllTokens()
					return
				}
				//                 log.Println("#### args", args, attr, args[len(args)-1])
				if attr == "write" {
					token, err = requestToken("write", args[len(args)-1])
					tokenKind = "write"
				} else {
					token, err = requestToken("read", args[len(args)-1])
					tokenKind = "read"
				}
				if err != nil {
					exit("unable to get valid token", err)
				}
				fmt.Println(token)
				if tokenKind == "write" {
					fmt.Println("\nSet CHESS_WRITE_TOKEN env variable with it to re-use in other commands")
				} else {
					fmt.Println("\nSet CHESS_TOKEN env variable with it to re-use in other commands")
				}
			} else {
				fmt.Println("ERROR")
			}
		},
	}
	return cmd
}
