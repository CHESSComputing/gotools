package cmd

// CHESComputing foxden tool: kerberos module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/credentials"
)

// helper function to return user and password
func userPassword() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

// https://github.com/jcmturner/gokrb5/issues/7
func kuserFromCache(cacheFile string) (*credentials.Credentials, error) {
	kfile := _srvConfig.Kerberos.Krb5Conf
	if kfile == "" {
		kfile = "/etc/krb5.conf"
	}
	cfg, err := config.Load(kfile)
	ccache, err := credentials.LoadCCache(cacheFile)
	client, err := client.NewClientFromCCache(ccache, cfg)
	err = client.Login()
	if err != nil {
		return nil, err
	}
	return client.Credentials, nil

}

func userTicket() (string, []byte) {
	// get user login/password
	user, password := userPassword()
	fname := fmt.Sprintf("krb5_%d_%v", os.Getuid(), time.Now().Unix())
	//     tmpFile, err := ioutil.TempFile("/tmp", fname)
	tmpFile, err := os.OpenFile(tempFilePath(fname), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		exit("Unable to get tmp file", err)
	}
	defer os.Remove(tmpFile.Name())

	cmd := exec.Command("kinit", "-c", tmpFile.Name(), user)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		msg := fmt.Sprintf("Fail to execute '%v'", cmd)
		exit(msg, err)
	}

	// start command execution
	err = cmd.Start()
	if err != nil {
		exit("unable to start kinit", err)
	}

	// write our input to our pipe for command
	io.WriteString(stdin, password)

	// explicitly close the input informing command that we're done
	stdin.Close()

	// wait for command to finish its execution
	cmd.Wait()

	// read tmp file content and return the ticket
	ticket, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil || len(ticket) == 0 {
		exit("unable to read kerberos credentials", err)
	}
	return user, ticket
}

// helper function to get kerberos ticket
func getKerberosTicket(krbFile string) (string, []byte) {
	if krbFile != "" {
		// read krbFile and check user credentials
		creds, err := kuserFromCache(krbFile)
		if err != nil {
			msg := fmt.Sprintf("\nUnable to get valid kerberos credentials from %s", krbFile)
			msg = fmt.Sprintf("%s\nPlease check that you have valid kerberos ticket, i.e. run kinit command", msg)
			msg = fmt.Sprintf("%s\nCheck your KRB5CCNAME environment to have FILE:/path/...", msg)
			msg = fmt.Sprintf("%s and use this file in --kfile option", msg)
			exit(msg, err)
		}
		if creds.Expired() {
			msg := fmt.Sprintf("user credentials are expired, please obtain new/valid kerberos file %s", krbFile)
			exit(msg, nil)
		}
		ticket, err := ioutil.ReadFile(krbFile)
		if err != nil {
			exit("unable to read kerberos credentials", err)
		}
		user := creds.UserName()
		return user, ticket
	}
	return userTicket()
}
