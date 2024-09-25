package cmd

// CHESComputing foxden tool: SpecScans data module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

// helper function to get specdata
// SpecScans represents SpecScans object returned from discovery service
type SpecScans struct {
	ID          string   `json:"id"`
	Site        string   `json:"site"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Bucket      string   `json:"bucket"`
}

// SpecScansRecord represents SpecScans record returned by discovery service
type SpecScansRecord struct {
	Status string      `json:"status"`
	Data   []SpecScans `json:"data"`
}

// helper function to fetch sites info from discovery service
func specdata(did string) SpecScansRecord {
	var results SpecScansRecord
	rurl := fmt.Sprintf("%s/spec/%s", _srvConfig.Services.SpecScansURL, did)
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

// helper function to get SpecScans data records
func getSpecScans(user, query string) ([]map[string]any, error) {
	// check if we got request from trusted client
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		// get trusted token and assign it to http write request
		if _httpReadRequest.Token == "" {
			if token, err := trustedUser(); err == nil {
				_httpReadRequest.Token = token
				defer func() {
					_httpReadRequest.Token = ""
				}()
			}
		}
	}

	var records []map[string]any
	if query == "" {
		query = "{}"
	}
	rec := services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query, Idx: 0, Limit: -1},
	}

	data, err := json.Marshal(rec)
	rurl := fmt.Sprintf("%s/search", _srvConfig.Services.SpecScansURL)
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("unable to fetch data from SpecScans data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from SpecScans data service", err)
	}
	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)

	//     var response services.ServiceResponse
	//     err = json.Unmarshal(data, &response)
	if err != nil {
		log.Println("response data", string(data))
		exit("Unable to unmarshal the data", err)
	}
	//     if response.HttpCode == 200 {
	//         records = response.Results.Records
	//     }
	records = response.Results.Records
	return records, nil
}

// helper function to provide usage of spec option
func specUsage() {
	fmt.Println("foxden spec <ls|view> [options]")
	fmt.Println("foxden spec add <file.json>")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all spec data records:")
	fmt.Println("foxden spec ls")
	fmt.Println("\n# list specific SpecScans data record:")
	fmt.Println("foxden spec view <DID>")
	fmt.Println("\n# add new SpecScans data record")
	fmt.Println("foxden spec add <file.json>")
	fmt.Println("\n# the same as above but provide json output")
	fmt.Println("foxden spec add <file.json> --json")
}

// helper function to add spec data record
func specAddRecord(args []string, jsonOutput bool) {
	if len(args) == 1 {
		fmt.Println("manual insertion is not implemented yet")
		specUsage()
		os.Exit(1)
	}
	// user must provide client spec add schema file.json
	fname := args[1]

	// check if we got request from trusted client
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		// get trusted token and assign it to http write request
		if _httpWriteRequest.Token == "" {
			if token, err := trustedUser(); err == nil {
				_httpWriteRequest.Token = token
				defer func() {
					_httpWriteRequest.Token = ""
				}()
			}
		}
	}

	// check if given fname is a file
	_, err := os.Stat(fname)
	exit(fmt.Sprintf("unable to check file stat, file %s", fname), err)
	file, err := os.Open(fname)
	exit(fmt.Sprintf("unable to open file %s", fname), err)
	defer file.Close()
	data, err := io.ReadAll(file)
	exit(fmt.Sprintf("unable to read file %s", fname), err)
	// Try to unmarshal the file's data as a single record first...
	var record map[string]any
	err = json.Unmarshal(data, &record)
	if err != nil {
		// If the file's data couldn't be unmarshalled to a single record, try to
		// unmarshal them as multiple records...
		var records []map[string]any
		err = json.Unmarshal(data, &records)
	}
	exit(fmt.Sprintf("unable to unmarshal data, file %s", fname), err)

	// add new SpecScans record
	rurl := fmt.Sprintf("%s/add", _srvConfig.Services.SpecScansURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	exit("unable to fetch data from SpecScans data service", err)
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	exit("unable to read data from SpecScans data service", err)

	if jsonOutput {
		fmt.Printf(string(data))
		return
	}
	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		log.Println("unable to Unarshal data into ServiceResponse, the response is %+s", string(data))
	}
	exit("Unable to unmarshal the data", err)
	if response.Status == "ok" || response.HttpCode == 200 {
		fmt.Printf("SUCCESS: record was successfully added with did, HTTP code 200\n")
	} else {
		// check if we got middleware error
		if response.SrvCode == 0 {
			fmt.Printf("SUCCESS: record was successfully added with did, service code 0\n")
		} else {
			fmt.Printf("ERROR: failed to add record to SpecScans service\n%+v", response.String())
		}
	}
}

// helper function to list SpecScans data records
func specListRecord(user, spec string, jsonOutput bool) {
	records, err := getSpecScans(user, spec)
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
		fmt.Printf("did        : %v\n", r["did"])
		fmt.Printf("schema     : %v\n", r["schema"])
		fmt.Printf("cycle      : %v\n", r["cycle"])
		fmt.Printf("beamline   : %v\n", r["beamline"])
		fmt.Printf("btr        : %v\n", r["btr"])
		fmt.Printf("sample_name: %v\n", r["sample_name"])
		//         fmt.Printf("%+v", r)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")

}

// helper function to print spec data records in Json format
func specJsonRecord(user, query string, jsonOutput bool) {
	//     query := fmt.Sprintf("{\"did\": \"%s\"}", did)
	//     query := "did:" + did
	records, err := getSpecScans(user, query)
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
}

func specCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "foxden SpecScans commands",
		Long:  "foxden SpecScans commands to access FOXDEN SpecScans service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			if len(args) == 0 {
				specUsage()
			} else if args[0] == "ls" {
				user, _ := getUserToken()
				if len(args) == 2 {
					specListRecord(user, args[1], jsonOutput)
				} else {
					specListRecord(user, "", jsonOutput)
				}
			} else if args[0] == "view" {
				user, _ := getUserToken()
				if len(args) == 2 {
					specJsonRecord(user, args[1], jsonOutput)
				} else {
					specJsonRecord(user, "", jsonOutput)
				}
			} else if args[0] == "add" {
				writeToken()
				specAddRecord(args, jsonOutput)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		specUsage()
		return nil
	})
	return cmd
}
