package cmd

// CHESComputing foxden tool: utils module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	authz "github.com/CHESSComputing/golib/authz"
	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	"github.com/CHESSComputing/golib/utils"
)

//go:embed static
var StaticFs embed.FS

var doc = "Complete documentation at https://chesscomputing.github.io/FOXDEN/"

var _httpReadRequest, _httpWriteRequest, _httpDeleteRequest *services.HttpRequest

type MapRecord map[string]any

var _jsonOutputError bool

// helper function to exit with message and error
func exit(msg string, err error) {
	if err != nil {
		verbose := strings.ToLower(fmt.Sprintf("%v", os.Getenv("FOXDEN_VERBOSE")))
		if verbose == "1" || verbose == "true" {
			log.Println(utils.Stack())
		}
		if _jsonOutputError {
			resp := services.ServiceResponse{
				Service:   "foxden CLI",
				Status:    msg,
				Timestamp: time.Now().String(),
				Error:     err.Error()}
			if data, err := json.Marshal(resp); err == nil {
				fmt.Println(string(data))
			}
			os.Exit(1)
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
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		return "", nil
	}
	tfile := fmt.Sprintf("%s/.foxden.read.token", os.Getenv("HOME"))
	var token string
	if _httpReadRequest.Token == "" {
		if os.Getenv("FOXDEN_TOKEN") != "" {
			token = utils.ReadToken(os.Getenv("FOXDEN_TOKEN"))
		} else {
			err := generateToken(tfile, "", 0)
			exit("Unable to generate access token", err)
			token = utils.ReadToken(tfile)
		}
		if token == "" {
			exit("Please obtain read access token and put it into FOXDEN_TOKEN env or file", nil)
		}
		_httpReadRequest.Token = token
	}
	return _httpReadRequest.Token, nil
}

// helper function to obtain write access token
func writeAccessToken() (string, error) {
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		return "", nil
	}
	tfile := fmt.Sprintf("%s/.foxden.write.token", os.Getenv("HOME"))
	var token string
	if _httpWriteRequest.Token == "" {
		if os.Getenv("FOXDEN_WRITE_TOKEN") != "" {
			token = utils.ReadToken(os.Getenv("FOXDEN_WRITE_TOKEN"))
		} else {
			err := generateToken(tfile, "", 0)
			exit("Unable to generate write token", err)
			token = utils.ReadToken(tfile)
		}
		if token == "" {
			exit("Please obtain read access token and put it into FOXDEN_WRITE_TOKEN env or file", nil)
		}
		_httpWriteRequest.Token = token
	}
	return _httpWriteRequest.Token, nil
}

// helper function to obtain delete access token
func deleteAccessToken() (string, error) {
	tfile := fmt.Sprintf("%s/.foxden.delete.token", os.Getenv("HOME"))
	var token string
	if _httpDeleteRequest.Token == "" {
		if os.Getenv("FOXDEN_DELETE_TOKEN") != "" {
			token = utils.ReadToken(os.Getenv("FOXDEN_DELETE_TOKEN"))
		} else {
			err := generateToken(tfile, "", 0)
			exit("Unable to generate write token", err)
			token = utils.ReadToken(tfile)
		}
		if token == "" {
			exit("Please obtain read access token and put it into FOXDEN_DELETE_TOKEN env or file", nil)
		}
		_httpDeleteRequest.Token = token
	}
	return _httpDeleteRequest.Token, nil
}

// helper function to get user and token
func getUserToken() (string, string) {
	token, err := accessToken()
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" && token == "" {
		return "trusted", ""
	}
	if err != nil {
		exit("unable to get access token", err)
	}
	claims, err := authz.TokenClaims(token, srvConfig.Config.Authz.ClientID)
	if err != nil {
		exit("unable to read token claims, please check FOXDEN_TOKEN env, and run 'foxden token view'", err)
	}
	user := claims.CustomClaims.User
	return user, token
}

// helper function to obtain write access token
func writeToken() (string, error) {
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		return "", nil
	}
	token, err := writeAccessToken()
	if _httpWriteRequest.Token == "" {
		token = utils.ReadToken(os.Getenv("FOXDEN_WRITE_TOKEN"))
		if token == "" {
			exit("Please obtain write access token and put it into FOXDEN_WRITE_TOKEN env or file", nil)
		}
		_, err = authz.TokenClaims(token, srvConfig.Config.Authz.ClientID)
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
		_, err := authz.TokenClaims(token, srvConfig.Config.Authz.ClientID)
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

func printRecords(records []MapRecord, show string) {
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
	out := []string{}
	for _, key := range keys {
		for _, rec := range records {
			if val, ok := rec[key]; ok {
				pad := strings.Repeat(" ", maxKey-len(key))
				out = append(out, fmt.Sprintf("%s%s: %v", key, pad, val))
			}
		}
	}
	out = utils.List2Set(out)
	sort.Strings(out)
	for _, item := range out {
		fmt.Println(item)
	}
}

// helper function to print given JSON file
func recordInfo(fname string) {
	fmt.Printf("### Example of %s\n", fname)
	// Open the file from the embedded file system
	file, err := StaticFs.Open(fmt.Sprintf("static/%s", fname))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the file content
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	// Print the content
	fmt.Println(string(data))
}
