package cmd

// CHESComputing foxden tool: utils module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
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

// GetSystemInfo returns the current user name, all non-loopback IPs, and all MAC addresses.
func GetSystemInfo() (string, []string, []string, error) {
	// Get current user
	u, err := user.Current()
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	username := u.Username

	// Collect all non-loopback IPv4 addresses
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return username, nil, nil, fmt.Errorf("failed to get IP addresses: %w", err)
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ips = append(ips, ipnet.IP.String())
		}
	}
	if len(ips) == 0 {
		ips = []string{"N/A"}
	}

	// Collect all MAC addresses from active interfaces
	var macs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return username, ips, nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && len(iface.HardwareAddr) > 0 {
			macs = append(macs, iface.HardwareAddr.String())
		}
	}
	if len(macs) == 0 {
		macs = []string{"N/A"}
	}

	return username, ips, macs, nil
}

// helper function to get trusted user name
func getTrustedUser() string {
	var trustedUser string
	user, ips, macs, err := GetSystemInfo()
	if err != nil {
		exit("unable to obtain system info for trusted user", err)
	}

	// to check system info we can either use TrustedUsers of server configuration
	// or, rely on FOXDEN Authz /trusted_client end-point
	if len(srvConfig.Config.TrustedUsers) == 0 {
		// use Authx /trusted_client end-point
		rec := make(map[string]any)
		rec["user"] = user
		rec["ips"] = ips
		rec["macs"] = macs
		data, err := json.Marshal(rec)
		if err != nil {
			exit("Unable to marshal system infor record", err)
		}
		rurl := fmt.Sprintf("%s/trusted_client", srvConfig.Config.AuthzURL)
		resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
		if err != nil {
			exit(fmt.Sprintf("fail to check trusted user info in FOXDEN Authz server, data=%v", string(data)), err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			exit(fmt.Sprintf("Unable to read response body %v", data), err)
		}
		var response services.ServiceResponse
		err = json.Unmarshal(data, &response)
		if err != nil {
			exit("Unable to unmarshal response body", err)
		}
		if response.SrvCode == 0 && response.Status == "ok" {
			trustedUser = user
		}
	} else {
		// rely on TrustedUsers configuration settings
		for _, tuser := range srvConfig.Config.TrustedUsers {
			for _, ip := range ips {
				for _, mac := range macs {
					if tuser.User == user && tuser.IP == ip && tuser.MAC == mac {
						trustedUser = tuser.User
						break
					}
				}
			}
		}
	}
	if trustedUser == "" {
		exit("No trusted user info found in FOXDEN", errors.New("auth failure"))
	}
	return trustedUser
}

// helper function to get user from the token
func getUserFromToken(token string) string {
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		return getTrustedUser()
	}
	claims, err := authz.TokenClaims(token, srvConfig.Config.Authz.ClientID)
	if err != nil {
		exit("unable to read token claims, please check FOXDEN_TOKEN env, and run 'foxden token view'", err)
	}
	user := claims.CustomClaims.User
	return user
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

// TrackTime provides elapsed time of function execution, e.g.
// add the following into any function you want to trace
// defer TrackTime("MyFunction", verbose)()
func TrackTime(name string, verboseTime bool) func() {
	start := time.Now()
	return func() {
		if verboseTime {
			fmt.Printf("%s took %v\n", name, time.Since(start))
		}
	}
}

// readJsonData reads input from a file or stdin if fname == "-"
func readJsonData(fname string) ([]byte, error) {
	var r io.Reader
	if fname == "-" {
		r = os.Stdin
	} else {
		_, err := os.Stat(fname)
		if err != nil {
			return nil, fmt.Errorf("cannot stat %q: %w", fname, err)
		}
		f, err := os.Open(fname)
		if err != nil {
			return nil, fmt.Errorf("cannot open %q: %w", fname, err)
		}
		defer f.Close()
		r = f
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}
	return data, nil
}
