package cmd

// CHESComputing foxden tool: meta-data module
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
	"net/http"
	"os"
	"strings"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// helper function to get metadata
// UserMetaData represents UserMetaData object returned from discovery service
type UserMetaData struct {
	ID          string   `json:"id"`
	Site        string   `json:"site"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Bucket      string   `json:"bucket"`
}

// UserMetaDataRecord represents UserMetaData record returned by discovery service
type UserMetaDataRecord struct {
	Status string         `json:"status"`
	Data   []UserMetaData `json:"data"`
}

// helper function to fetch sites info from discovery service
func userMetadata(site string) UserMetaDataRecord {
	var results UserMetaDataRecord
	rurl := fmt.Sprintf("%s/meta/%s", srvConfig.Config.Services.UserMetaDataURL, site)
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}
	resp, err := http.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		return results
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&results); err != nil {
		fmt.Println("ERROR:", err)
		return results
	}
	return results
}

// helper function to get meta-data records
func getUserMeta(user, query string, skeys []string, sorder, idx, limit int) ([]map[string]any, int, error) {
	// check if we got request from trusted client
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		// get trusted token and assign it to http write request
		if token, err := trustedUser(); err == nil {
			_httpReadRequest.Token = token
			defer func() {
				_httpReadRequest.Token = ""
			}()
		}
	}
	var records []map[string]any
	if query == "" {
		query = "{}"
	}
	rec := services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query, Idx: idx, Limit: limit, SortKeys: skeys, SortOrder: sorder},
	}

	data, err := json.Marshal(rec)
	rurl := fmt.Sprintf("%s/search", srvConfig.Config.Services.UserMetaDataURL)
	if os.Getenv("FOXDEN_VERBOSE") != "" {
		fmt.Println("FOXDEN query:", rurl)
	}
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("fail /search, unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from meta-data service", err)
	}
	err = json.Unmarshal(data, &records)

	//     var response services.ServiceResponse
	//     err = json.Unmarshal(data, &response)
	if err != nil {
		log.Println("response data", string(data))
		exit("Unable to unmarshal the data", err)
	}

	// get total number of records
	rec = services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query},
	}

	data, err = json.Marshal(rec)
	rurl = fmt.Sprintf("%s/count", srvConfig.Config.Services.UserMetaDataURL)
	resp, err = _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("fail /count, unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from meta-data service", err)
	}
	var nrecords int
	err = json.Unmarshal(data, &nrecords)
	if err != nil {
		exit("unable to unmarshal the data", err)
	}
	return records, nrecords, nil
}

func didUserMetaData() (string, string, string) {
	attrs := strings.Join(utils.DIDKeys(""), ",")
	sep := "/"
	div := "="
	if srvConfig.Config != nil {
		if srvConfig.Config.DID.Attributes != "" {
			attrs = srvConfig.Config.DID.Attributes
		}
		if srvConfig.Config.DID.Separator != "" {
			sep = srvConfig.Config.DID.Separator
		}
		if srvConfig.Config.DID.Divider != "" {
			div = srvConfig.Config.DID.Divider
		}
	}
	return attrs, sep, div
}

// helper function to provide usage of meta option
func userMetaUsage() {
	fmt.Println("foxden umeta <ls|rm|view> [options]")
	fmt.Println("foxden umeta <add|amend> <file.json> {options}")
	fmt.Println("options: --did-div=<divider> --json --elapsed-time")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list user metadata records:")
	fmt.Println("foxden umeta ls")
	fmt.Println("\n# list user metadata records for specific range:")
	fmt.Println("foxden umeta ls --idx=10 --limit=20")
	fmt.Println("\n# list all user metadata records using specific sorting key(s) and order:")
	fmt.Println("foxden umeta ls --sort-keys=date --sort-order=1")
	fmt.Println("\n# list specific user metadata record:")
	fmt.Println("foxden umeta view <DID>")
	fmt.Println("\n# remove user metadata record:")
	fmt.Println("foxden umeta rm <DID>")
	fmt.Println("\n# add user metadata record:")
	fmt.Println("foxden umeta add <file.json>")
	fmt.Println("\n# amend user metadata record")
	fmt.Println("foxden umeta amend <file.json>")
}

// helper function to add meta data record
func userMetaAddRecord(user string, data []byte, jsonOutput bool, update bool, elapsedTime bool) {
	defer TrackTime(elapsedTime)()
	// check if we got request from trusted client
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		// get trusted token and assign it to http write request
		if token, err := trustedUser(); err == nil {
			_httpWriteRequest.Token = token
			defer func() {
				_httpWriteRequest.Token = ""
			}()
		}
	}

	var record map[string]any
	err := json.Unmarshal(data, &record)
	exit("unable to unmarshal data", err)

	// add proper did
	did, ok := record["did"]
	if !ok || did == "" {
		log.Fatal("provided metadata record does not contain did")
	}

	// add user info to metadata record
	if _, ok := record["user"]; !ok {
		record["user"] = user
	}

	data, err = json.MarshalIndent(record, "", "  ")
	exit("unable to marshal data", err)
	rurl := fmt.Sprintf("%s/record", srvConfig.Config.Services.UserMetaDataURL)
	var resp *http.Response
	if update {
		resp, err = _httpWriteRequest.Put(rurl, "application/json", bytes.NewBuffer(data))
	} else {
		resp, err = _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	}
	msg := fmt.Sprintf("fail %s unable to fetch data from meta-data service", rurl)
	exit(msg, err)
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	exit("unable to read data from meta-data service", err)

	if jsonOutput {
		fmt.Printf(string(data))
		return
	}
	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)
	exit("Unable to unmarshal the data", err)
	if response.Status == "ok" {
		fmt.Printf("SUCCESS: record was successfully added with did\n")
		fmt.Println(did)
	} else {
		// check if we got middleware error
		if response.HttpCode == 0 {
			fmt.Printf("ERROR: %s", string(data))
		} else {
			fmt.Printf("ERROR: failed to add record to UserMetaData service\n%+v", response.String())
		}
	}
}

// helper function to delete meta-data record
func userMetaDeleteRecord(user, did string, jsonOutput bool, elapsedTime bool) {
	defer TrackTime(elapsedTime)()
	if did == "" {
		userMetaUsage()
		os.Exit(1)
	}
	token, err := deleteAccessToken()
	exit("", err)
	rurl := fmt.Sprintf("%s/record?did=%s&user=%s", srvConfig.Config.Services.UserMetaDataURL, did, user)
	req, err := http.NewRequest("DELETE", rurl, nil)
	exit("", err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	exit("", err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	exit("", err)
	var response services.ServiceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("ERROR", err, "response body", string(body))
		os.Exit(1)
	}
	if jsonOutput {
		fmt.Println(string(body))
		return
	}
	if response.Status == "ok" {
		fmt.Printf("SUCCESS: record %s was successfully removed\n", did)
	} else {
		fmt.Printf("WARNING: record %s failed to be removed\n", did)
	}

}

// helper funtion to list meta-data records
func userMetaListRecord(user, spec string, skeys []string, sorder, idx, limit int, jsonOutput, elapsedTime bool) {
	defer TrackTime(elapsedTime)()
	rurl := srvConfig.Config.UserMetaDataURL
	records, nrecords, err := getMeta(rurl, user, spec, skeys, sorder, idx, limit)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	if jsonOutput {
		if data, err := json.MarshalIndent(records, "", " "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Println("ERROR", err)
			os.Exit(1)
		}
		return
	}

	for _, r := range records {
		fmt.Println("---")
		if data, err := json.MarshalIndent(r, "", "   "); err == nil {
			fmt.Printf("%s\n", string(data))
		} else {
			fmt.Printf("%+v\n", r)
		}
	}
	fmt.Println("---")
	fmt.Printf("Showing %d-%d out of %d records, for more records use --idx/--limit options\n", idx, idx+limit, nrecords)
}

// helper function to print meta data records in Json format
func userMetaJsonRecord(user, did string, skeys []string, sorder, idx, limit int, jsonOutput, elapsedTime bool) {
	defer TrackTime(elapsedTime)()
	query := "did:" + did
	if verbose > 0 {
		log.Println("### query", query)
	}
	rurl := srvConfig.Config.UserMetaDataURL
	records, nrecords, err := getMeta(rurl, user, query, skeys, sorder, idx, limit)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	if jsonOutput {
		if data, err := json.MarshalIndent(records, "", " "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Println("ERROR", err)
			os.Exit(1)
		}
		return
	}
	for _, r := range records {
		fmt.Println("---")
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			exit("unable to marshal data", err)
		}
		fmt.Println(string(data))
	}
	fmt.Println("---")
	if verbose > 0 {
		fmt.Printf("Showing %d-%d out of %d records, for more records use --idx/--limit options\n", idx, idx+limit, nrecords)
	}
}

func userMetaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "umeta",
		Short: "foxden user MetaData commands",
		Long:  "foxden user MetaData commands to access FOXDEN user metadata service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			sortKeys, _ := cmd.Flags().GetString("sort-keys")
			sortOrder, _ := cmd.Flags().GetInt("sort-order")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			elapsedTime, _ := cmd.Flags().GetBool("elapsed-time")
			idx, _ := cmd.Flags().GetInt("idx")
			limit, _ := cmd.Flags().GetInt("limit")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			var skeys []string
			if sortKeys != "" {
				for _, k := range strings.Split(sortKeys, ",") {
					skeys = append(skeys, k)
				}
			}
			if sortOrder == 0 {
				sortOrder = -1
			}
			if len(args) == 0 {
				userMetaUsage()
			} else if args[0] == "ls" {
				user, _ := getUserToken()
				if len(args) == 2 {
					userMetaListRecord(user, args[1], skeys, sortOrder, idx, limit, jsonOutput, elapsedTime)
				} else {
					userMetaListRecord(user, "", skeys, sortOrder, idx, limit, jsonOutput, elapsedTime)
				}
			} else if args[0] == "view" {
				user, _ := getUserToken()
				if len(args) == 2 {
					userMetaJsonRecord(user, args[1], skeys, sortOrder, idx, limit, jsonOutput, elapsedTime)
				} else {
					userMetaJsonRecord(user, "", skeys, sortOrder, idx, limit, jsonOutput, elapsedTime)
				}
			} else if args[0] == "add" {
				token, _ := writeToken()
				var fname string
				if len(args) == 2 {
					fname = args[1]
				} else {
					userMetaUsage()
					exit("please provide <file.json>", errors.New("no input file"))
				}
				user := getUserFromToken(token)
				data, err := readJsonData(fname)
				exit("unable to read data from input file", err)
				userMetaAddRecord(user, data, jsonOutput, false, elapsedTime)
			} else if args[0] == "amend" {
				token, _ := writeToken()
				var fname string
				if len(args) == 2 {
					fname = args[1]
				} else {
					userMetaUsage()
					exit("please provide <file.json>", errors.New("no input file"))
				}
				user := getUserFromToken(token)
				data, err := readJsonData(fname)
				exit("unable to read data from input file", err)
				userMetaAddRecord(user, data, jsonOutput, true, elapsedTime)
			} else if args[0] == "rm" {
				deleteToken()
				user, _ := getUserToken()
				if user == "" {
					exit("unable to get user name from token value", errors.New("unknown user"))
				}
				if len(args) != 2 {
					userMetaUsage()
					exit("please provide did", errors.New("no did"))
				}
				did := args[1]
				metaDeleteRecord(user, did, jsonOutput, elapsedTime)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().Bool("elapsed-time", false, "print out elapsed time")
	cmd.PersistentFlags().String("sort-keys", "date", "sort key(s), if multiple keys separate them by comma (default: date)")
	cmd.PersistentFlags().Int("sort-order", -1, "sort order: 1 ascending, -1 desecnding (default)")
	cmd.PersistentFlags().Int("idx", 0, "start index, default 0")
	cmd.PersistentFlags().Int("limit", 100, "limit number of records to given value, default 100")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		userMetaUsage()
		return nil
	})
	return cmd
}
