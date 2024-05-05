package cmd

// CHESComputing foxden tool: auth module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	authz "github.com/CHESSComputing/golib/authz"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

var envTokens []string = []string{"FOXDEN_TOKEN", "FOXDEN_WRITE_TOKEN", "FOXDEN_DELETE_TOKEN"}

// helper function to provide usage of meta option
func authUsage() string {
	var out string
	out += fmt.Sprintf("\nfoxden token create <scope: read|write|delete> [options]\n")
	out += fmt.Sprintf("foxden token view [options]\n\n")
	out += fmt.Sprintf("options: --kfile=keytab, --token=<token or file>\n")
	out += fmt.Sprintf("defaults: token generated with read scope\n")
	out += fmt.Sprintf("          kfile is /tmp/krb5cc_<UID>\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("Examples: \n")
	out += fmt.Sprintf("# generate read token\n")
	out += fmt.Sprintf("foxden token create read\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate read token from specific /path/keytab file\n")
	out += fmt.Sprintf("foxden token create read --kfile=/path/keytab\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate write token\n")
	out += fmt.Sprintf("foxden token create write\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# view provided token=abc...xyz\n")
	out += fmt.Sprintf("foxden token view --token=abc...xyz\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# view existing token stored in /tmp/token file\n")
	out += fmt.Sprintf("foxden token view --token=/tmp/token\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# view existing token stored in %s\n", envTokens)
	out += fmt.Sprintf("foxden token view\n")
	return out
}

// helper function to return user's key file name
func keyFile() string {
	u, err := user.Current()
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	return tempFilePath(fmt.Sprintf("krb5cc_%s", u.Uid))
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
	printMap(response)
	return token, errors.New(msg)
}

// helper function to generate access token
func generateToken(keyFileName string) error {
	// check if user has default read token
	fname := fmt.Sprintf("%s/.foxden.access", os.Getenv("HOME"))
	if _, err := os.Stat(fname); err == nil {
		// file exists, let's read token and check its validity
		file, err := os.Open(fname)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		token := string(data)
		claims, err := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
		rclaims := claims.RegisteredClaims
		etime := rclaims.ExpiresAt
		if etime.After(time.Now()) {
			return nil
		}
	}

	// if user does not have default foxden.access valid token file we'll proceed with
	// getting user's kerberos credentials
	var kfile string

	// expand tilde in keyFileName if it exists
	if strings.HasPrefix(keyFileName, "~/") {
		usr, _ := user.Current()
		dir := usr.HomeDir
		keyFileName = filepath.Join(dir, keyFileName[2:])
	}

	// check if we have provided with valid kerberos file name
	if _, err := os.Stat(keyFileName); err == nil {
		kfile = keyFileName
	} else {
		// check if user has kerberos file in place, i.e. /tmp/krb5cc_<uid>
		kfile = keyFile()
		if _, err := os.Stat(kfile); os.IsNotExist(err) {
			fmt.Printf("No kerberos ticket file %s found, please run:\n", kfile)
			fmt.Printf("# in (ba)sh environment, export KRB5CCNAME=FILE:%s\n", kfile)
			fmt.Printf("# in (t)csh environment, setenv KRB5CCNAME FILE:%s\n", kfile)
			fmt.Println("kinit")
			fmt.Println("")
			return err
		}
	}

	token, err := requestToken("read", kfile)
	if err != nil {
		return err
	}
	file, err := os.Create(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	file.Write([]byte(token))
	return nil
}

func inspectAllTokens(tkn string) {
	var token string
	if tkn != "" {
		token = utils.ReadToken(tkn)
		inspectToken(token)
		return
	}
	tfile := fmt.Sprintf("%s/.foxden.access", os.Getenv("HOME"))
	if _, err := os.Stat(tfile); err == nil {
		token = utils.ReadToken(tfile)
		fmt.Println(tfile)
		inspectToken(token)
	} else {
		fmt.Println("No input token is provided, will lookup them from env: FOXDEN_TOKEN | FOXDEN_WRITE_TOKEN | FOXDEN_DELETE_TOKEN...")
	}
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
		Use:   "token",
		Short: "foxden token commands",
		Long:  "foxden token commands: valid token is required to access FOXDEN services\n" + doc + "\n" + authUsage(),
		Run: func(cmd *cobra.Command, args []string) {
			tkn, _ := cmd.Flags().GetString("token")
			kfile, _ := cmd.Flags().GetString("kfile")
			if len(args) == 0 {
				authUsage()
			} else if args[0] == "view" {
				inspectAllTokens(tkn)
			} else if args[0] == "create" {
				attr := "read"
				if len(args) > 1 {
					attr = args[1]
				}
				var token, tokenKind string
				tokenEnv := "FOXDEN_TOKEN"
				var err error
				if attr == "write" {
					tokenKind = "write"
					tokenEnv = "FOXDEN_WRITE_TOKEN"
				} else if attr == "delete" {
					tokenKind = "delete"
					tokenEnv = "FOXDEN_DELETE_TOKEN"
				} else {
					tokenKind = attr
				}
				if tokenKind == "read" {
					err := generateToken(kfile)
					exit("unable to generate user access token", err)
					return
				}
				token, err = requestToken(tokenKind, kfile)
				if err != nil {
					exit("unable to get valid token", err)
				}
				fmt.Println(token)
				fmt.Printf("\nSet %s env variable with this token to re-use it in other commands\n", tokenEnv)
			} else {
				fmt.Println("ERROR: wrong argument(s), please see --help")
			}
		},
	}
	cmd.PersistentFlags().String("kfile", "", "Kerberos file to use")
	cmd.PersistentFlags().String("token", "", "token file or token string")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		authUsage()
		return nil
	})
	return cmd
}
