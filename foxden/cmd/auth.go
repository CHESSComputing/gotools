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
	srvConfig "github.com/CHESSComputing/golib/config"
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
	out += fmt.Sprintf("options: --kfile=keytab\n")
	out += fmt.Sprintf("         --token=<token or file>\n")
	out += fmt.Sprintf("         --ofile=<output fila name>\n")
	out += fmt.Sprintf("         --expires=<seconds>\n")
	out += fmt.Sprintf("defaults: token generated with read scope\n")
	out += fmt.Sprintf("          kfile is /tmp/krb5cc_<UID>\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("Examples: \n")
	out += fmt.Sprintf("# generate read token\n")
	out += fmt.Sprintf("foxden token create read\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate read token with long expiration, e.g. 6 hours (21600 seconds)\n")
	out += fmt.Sprintf("foxden token create read --expires=21600\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate read token from specific /path/keytab file and store it to $HOME/.foxden.read.token\n")
	out += fmt.Sprintf("foxden token create read --kfile=/path/keytab\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate read token from specific /path/keytab file and store it to token.read\n")
	out += fmt.Sprintf("foxden token create read --kfile=/path/keytab --ofile=token.read\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate write token, it will be stored to $HOME/.foxden.write.token\n")
	out += fmt.Sprintf("foxden token create write\n")
	out += fmt.Sprintf("\n")
	out += fmt.Sprintf("# generate write token and store it to token.write\n")
	out += fmt.Sprintf("foxden token create write --ofile=token.write\n")
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
	keyfile := strings.Replace(os.Getenv("KRB5CCNAME"), "FILE:", "", -1)
	if keyfile != "" {
		return keyfile
	}
	u, err := user.Current()
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	return tempFilePath(fmt.Sprintf("krb5cc_%s", u.Uid))
}

func requestToken(scope, kfile string, expires int) (string, error) {
	if kfile == "" {
		kfile = keyFile()
	}
	if os.Getenv("FOXDEN_DEBUG") != "" {
		fmt.Println("request token from", kfile)
	}
	var token string
	user, ticket := getKerberosTicket(kfile)
	rec := authz.Kerberos{
		User:    user,
		Scope:   scope,
		Ticket:  ticket,
		Expires: int64(expires),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return token, err
	}
	httpMgr := services.NewHttpRequest(scope, 1)
	rurl := fmt.Sprintf("%s/oauth/authorize", srvConfig.Config.Services.AuthzURL)
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
func generateToken(fname, keyFileName string, expires int) error {
	if os.Getenv("FOXDEN_DEBUG") != "" {
		fmt.Println("generate token from", fname)
	}
	// check if user has default read token
	if _, err := os.Stat(fname); err == nil {
		// file exists, let's read token and check its validity
		file, err := os.Open(fname)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		token := string(data)
		claims, err := authz.TokenClaims(token, srvConfig.Config.Authz.ClientID)
		rclaims := claims.RegisteredClaims
		etime := rclaims.ExpiresAt
		if etime.After(time.Now()) {
			return nil
		}
	}

	// if user does not have default foxden.read.token valid token file we'll proceed with
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

	token, err := requestToken("read", kfile, expires)
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
	rfile := fmt.Sprintf("%s/.foxden.read.token", os.Getenv("HOME"))
	wfile := fmt.Sprintf("%s/.foxden.write.token", os.Getenv("HOME"))
	dfile := fmt.Sprintf("%s/.foxden.delete.token", os.Getenv("HOME"))
	found := false
	for _, tfile := range []string{rfile, wfile, dfile} {
		if _, err := os.Stat(tfile); os.IsNotExist(err) {
			continue
		}
		token = utils.ReadToken(tfile)
		if token != "" {
			fmt.Println("")
			fmt.Println(tfile)
			inspectToken(token)
			found = true
		}
	}
	for _, env := range envTokens {
		token = utils.ReadToken(os.Getenv(env))
		if token != "" {
			s := fmt.Sprintf("%s: %s", env, token)
			fmt.Println("")
			fmt.Println(s)
			inspectToken(token)
			found = true
		}
	}
	if !found {
		fmt.Println("No input token is provided, will lookup them from env: FOXDEN_TOKEN | FOXDEN_WRITE_TOKEN | FOXDEN_DELETE_TOKEN...")
	}
}

func inspectToken(token string) {
	claims, err := authz.TokenClaims(token, srvConfig.Config.Authz.ClientID)
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

// helper function to get token as trusted user
func trustedUser() (string, error) {
	var token string
	scope := "write"

	// prepare trusted client data
	t := utils.NewTrustedClient()
	salt := authz.ReadSecret(srvConfig.Config.Encryption.Secret)
	edata, err := t.Encrypt(salt)

	// send encrypted data back to Authz server
	httpMgr := services.NewHttpRequest(scope, 1)
	ctype := "applicatin/octet-stream"
	buf := bytes.NewBuffer(edata)
	rurl := fmt.Sprintf("%s/oauth/trusted", srvConfig.Config.Services.AuthzURL)
	resp, err := httpMgr.Post(rurl, ctype, buf)
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}
	if resp.StatusCode != 200 {
		var sresp services.ServiceResponse
		err = json.Unmarshal(data, &sresp)
		if err != nil {
			return "", errors.New(string(data))
		}
		return "", errors.New(sresp.Error)
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
	return token, nil
}

func authCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "foxden token commands",
		Long:  "foxden token commands: valid token is required to access FOXDEN services\n" + doc + "\n" + authUsage(),
		Run: func(cmd *cobra.Command, args []string) {
			tkn, _ := cmd.Flags().GetString("token")
			kfile, _ := cmd.Flags().GetString("kfile")
			ofile, _ := cmd.Flags().GetString("ofile")
			expires, _ := cmd.Flags().GetInt("expires")
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
					fname := fmt.Sprintf("%s/.foxden.read.token", os.Getenv("HOME"))
					if ofile != "" {
						fname = ofile
					}
					err := generateToken(fname, kfile, expires)
					exit("unable to generate user access token", err)
					return
				}
				token, err = requestToken(tokenKind, kfile, expires)
				if err != nil {
					exit("unable to get valid token", err)
				}
				if tokenKind == "write" {
					fname := fmt.Sprintf("%s/.foxden.write.token", os.Getenv("HOME"))
					if ofile != "" {
						fname = ofile
					}
					file, err := os.Create(fname)
					if err != nil {
						log.Fatal(err)
					}
					defer file.Close()
					file.Write([]byte(token))
				} else if tokenKind == "delete" {
					fname := fmt.Sprintf("%s/.foxden.delete.token", os.Getenv("HOME"))
					if ofile != "" {
						fname = ofile
					}
					file, err := os.Create(fname)
					if err != nil {
						log.Fatal(err)
					}
					defer file.Close()
					file.Write([]byte(token))
				} else {
					fmt.Println(token)
					fmt.Printf("\nSet %s env variable with this token to re-use it in other commands\n", tokenEnv)
				}
			} else {
				fmt.Println("ERROR: wrong argument(s), please see --help")
			}
		},
	}
	cmd.PersistentFlags().String("kfile", "", "Kerberos file to use")
	cmd.PersistentFlags().String("token", "", "token file or token string")
	cmd.PersistentFlags().String("ofile", "", "output file to write to")
	cmd.PersistentFlags().Int("expires", 3600, "token expiration in seconds (default 1h)")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		authUsage()
		return nil
	})
	return cmd
}
