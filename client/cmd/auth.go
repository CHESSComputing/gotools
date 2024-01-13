package cmd

// CHESComputing client tool: auth module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"

	authz "github.com/CHESSComputing/golib/authz"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

var envTokens []string = []string{"CHESS_TOKEN", "CHESS_WRITE_TOKEN", "CHESS_DELETE_TOKEN"}

// helper function to provide usage of meta option
func authUsage() string {
	var out string
	out += fmt.Sprintf("client auth token <scope: read|write|delete> <--kfile=keytab>\n")
	out += fmt.Sprintf("client auth view <--token=token or file>\n\n")
	out += fmt.Sprintf("defaults: token generated with read scope\n")
	out += fmt.Sprintf("          kfile is /tmp/krb5cc_<UID>\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("Examples: \n")
	out += fmt.Sprintf("# generate read token\n")
	out += fmt.Sprintf("client auth token read\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate read token from specific /path/keytab file\n")
	out += fmt.Sprintf("client auth token read -kfile=/path/keytab\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate write token\n")
	out += fmt.Sprintf("client auth token write\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# view provided token=abc...xyz\n")
	out += fmt.Sprintf("client auth token view --token=abc...xyz\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# view existing token stored in /tmp/token file\n")
	out += fmt.Sprintf("client auth token view --token=/tmp/token\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# view existing token stored in %s\n", envTokens)
	out += fmt.Sprintf("client auth token view\n")
	return out
}

// helper function to return user's key file name
func keyFile() string {
	u, err := user.Current()
	if err != nil {
		fmt.Println("ERROR: ", err)
		os.Exit(1)
	}
	return fmt.Sprintf("/tmp/krb5cc_%s", u.Uid)
}

func requestToken(scope, fname string) (string, error) {
	if fname == "" {
		fname = keyFile()
	}
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

func inspectAllTokens(tkn string) {
	var token string
	if tkn != "" {
		token = utils.ReadToken(tkn)
		inspectToken(token)
		return
	}
	fmt.Println("No input token is provided, will lookup them from env...")
	for _, env := range envTokens {
		token = utils.ReadToken(os.Getenv(env))
		if token != "" {
			s := fmt.Sprintf("%s: %s", env, token)
			fmt.Println("")
			fmt.Println(s)
			inspectToken(token)
		}
	}
}

func inspectToken(token string) {
	claims, err := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
	rclaims := claims.RegisteredClaims
	fmt.Println()
	if err != nil {
		fmt.Println("ERROR        : ", err)
	}
	fmt.Println("AccessToken  : ", token)
	fmt.Println("Issuer       : ", rclaims.Issuer)
	fmt.Println("Subject      : ", rclaims.Subject)
	fmt.Println("Audience     : ", rclaims.Audience)
	fmt.Println("ExpiresAt    : ", rclaims.ExpiresAt)
	fmt.Println("Custom Claims: ", claims.CustomClaims.String())
}

func authCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "client authentication/authorization commands",
		Long:  "client authentication/authorization commands\n" + doc + "\n" + authUsage(),
		Run: func(cmd *cobra.Command, args []string) {
			tkn, _ := cmd.Flags().GetString("token")
			fname, _ := cmd.Flags().GetString("kfile")
			if len(args) == 0 {
				fmt.Print(authUsage())
			} else if args[0] == "token" {
				attr := args[1]
				var token, tokenKind string
				tokenEnv := "CHESS_TOKEN"
				var err error
				if attr == "view" {
					inspectAllTokens(tkn)
					return
				}
				if attr == "write" {
					tokenKind = "write"
					tokenEnv = "CHESS_WRITE_TOKEN"
				} else if attr == "delete" {
					tokenKind = "delete"
					tokenEnv = "CHESS_DELETE_TOKEN"
				} else {
					tokenKind = "read"
				}
				token, err = requestToken(tokenKind, fname)
				if err != nil {
					exit("unable to get valid token", err)
				}
				fmt.Println(token)
				fmt.Printf("\nSet %s env variable with it to re-use in other commands", tokenEnv)
			} else {
				fmt.Println("ERROR")
			}
		},
	}
	cmd.PersistentFlags().String("kfile", "", "Kerberos file to use")
	cmd.PersistentFlags().String("token", "", "token file or token string")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Println(authUsage())
		return nil
	})
	return cmd
}
