package cmd

// CHESComputing foxden tool: utils module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	authz "github.com/CHESSComputing/golib/authz"
	services "github.com/CHESSComputing/golib/services"
	"github.com/CHESSComputing/golib/utils"
)

var doc = "Complete documentation at https://foxden.classe.cornell.edu:8344/docs"

var _httpReadRequest, _httpWriteRequest, _httpDeleteRequest *services.HttpRequest

type MapRecord map[string]any

// helper function to exit with message and error
func exit(msg string, err error) {
	if err != nil {
		verbose := strings.ToLower(fmt.Sprintf("%v", os.Getenv("FOXDEN_VERBOSE")))
		if verbose == "1" || verbose == "true" {
			log.Println(utils.Stack())
		}
		reason := fmt.Sprintf("\n\nReason: %s", msg)
		log.Fatal("ERROR: ", err, reason)
	}
}

// helper function to properly provide fila path in tmp|TEMP areas on UNIX|Windows systems, respectively
func tempFilePath(fname string) string {
	var filePath string
	switch goos := runtime.GOOS; goos {
	case "windows":
		filePath = filepath.Join(os.Getenv("TEMP"), fname)
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		filePath = filepath.Join("/tmp", fname)
	default:
		log.Fatal("Unsupported operating system ", goos)
	}
	return filePath
}

// helper function to obtain read access token
func accessToken() (string, error) {
	var token string
	if _httpReadRequest.Token == "" {
		if os.Getenv("FOXDEN_TOKEN") != "" {
			token = utils.ReadToken(os.Getenv("FOXDEN_TOKEN"))
		} else {
			err := generateToken("")
			exit("Unable to generate access token", err)
			tfile := fmt.Sprintf("%s/.foxden.access", os.Getenv("HOME"))
			token = utils.ReadToken(tfile)
		}
		if token == "" {
			exit("Please obtain read access token and put it into FOXDEN_TOKEN env or file", nil)
		}
		_httpReadRequest.Token = token
	}
	return _httpReadRequest.Token, nil
}

// helper function to get user and token
func getUserToken() (string, string) {
	token, err := accessToken()
	if err != nil {
		exit("unable to get access token", err)
	}
	claims, err := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
	if err != nil {
		exit("unable to read token claims, please check FOXDEN_TOKEN env, and run 'foxden token view'", err)
	}
	rclaims := claims.RegisteredClaims
	user := rclaims.Subject
	return user, token
}

// helper function to obtain write access token
func writeToken() (string, error) {
	if _httpWriteRequest.Token == "" {
		token := utils.ReadToken(os.Getenv("FOXDEN_WRITE_TOKEN"))
		if token == "" {
			exit("Please obtain write access token and put it into FOXDEN_WRITE_TOKEN env or file", nil)
		}
		_, err := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
		if err != nil {
			exit("unable to use write token claims\nPlease check FOXDEN_WRITE_TOKEN env and set it up with token from 'foxden token create write' command", err)
		}
		_httpWriteRequest.Token = token
	}
	return _httpWriteRequest.Token, nil
}

// helper function to obtain delete access token
func deleteToken() (string, error) {
	if _httpDeleteRequest.Token == "" {
		token := utils.ReadToken(os.Getenv("FOXDEN_DELETE_TOKEN"))
		if token == "" {
			exit("Please obtain delete access token and put it into FOXDEN_DELETE_TOKEN env or file", nil)
		}
		_, err := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
		if err != nil {
			exit("unable to use delete token claims\nPlease check FOXDEN_DELETE_TOKEN env and set it up with token from 'foxden token create delete' command", err)
		}
		_httpDeleteRequest.Token = token
	}
	return _httpDeleteRequest.Token, nil
}

// helper function to print map record
func printRecord(rec MapRecord, sep string) {
	if sep != "" {
		fmt.Println(sep)
	}
	maxKey := 0
	for key, _ := range rec {
		if len(key) > maxKey {
			maxKey = len(key)
		}
	}
	keys := utils.MapKeys(rec)
	sort.Strings(keys)
	for _, key := range keys {
		val, _ := rec[key]
		pad := strings.Repeat(" ", maxKey-len(key))
		fmt.Printf("%s%s\t%v\n", key, pad, val)
	}
}

func printRecords(records []MapRecord) {
	// look-up all keys to get proper padding
	var keys []string
	for _, rec := range records {
		for _, key := range utils.MapKeys(rec) {
			keys = append(keys, key)
		}
	}
	keys = utils.List2Set(keys)
	sort.Strings(keys)
	maxKey := 0
	for _, key := range keys {
		if len(key) > maxKey {
			maxKey = len(key)
		}
	}
	for _, key := range keys {
		for _, rec := range records {
			if val, ok := rec[key]; ok {
				pad := strings.Repeat(" ", maxKey-len(key))
				fmt.Printf("%s%s\t: %v\n", key, pad, val)
			}
		}
	}
}
