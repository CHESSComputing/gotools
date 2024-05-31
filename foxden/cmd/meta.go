package cmd

// CHESComputing foxden tool: meta-data module
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
	"strings"

	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// helper function to get metadata
// MetaData represents MetaData object returned from discovery service
type MetaData struct {
	ID          string   `json:"id"`
	Site        string   `json:"site"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Bucket      string   `json:"bucket"`
}

// MetaDataRecord represents MetaData record returned by discovery service
type MetaDataRecord struct {
	Status string     `json:"status"`
	Data   []MetaData `json:"data"`
}

// helper function to fetch sites info from discovery service
func metadata(site string) MetaDataRecord {
	var results MetaDataRecord
	rurl := fmt.Sprintf("%s/meta/%s", _srvConfig.Services.MetaDataURL, site)
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
func getMeta(user, query string) ([]map[string]any, error) {
	var records []map[string]any
	if query == "" {
		query = "{}"
	}
	rec := services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query, Idx: 0, Limit: -1},
	}

	data, err := json.Marshal(rec)
	rurl := fmt.Sprintf("%s/search", _srvConfig.Services.MetaDataURL)
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("unable to fetch data from meta-data service", err)
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
	//     if response.HttpCode == 200 {
	//         records = response.Results.Records
	//     }
	return records, nil
}

func didMetaData() (string, string, string) {
	attrs := strings.Join(utils.DIDKeys(""), ",")
	sep := "/"
	div := "="
	if _srvConfig != nil {
		if _srvConfig.DID.Attributes != "" {
			attrs = _srvConfig.DID.Attributes
		}
		if _srvConfig.DID.Separator != "" {
			sep = _srvConfig.DID.Separator
		}
		if _srvConfig.DID.Divider != "" {
			div = _srvConfig.DID.Divider
		}
	}
	return attrs, sep, div
}

// helper function to provide usage of meta option
func metaUsage() {
	attrs, sep, div := didMetaData()
	fmt.Println("foxden meta <ls|rm|view> [options]")
	fmt.Println("foxden meta add <schema> <file.json> {options}")
	fmt.Println("options: --did-attrs=<attrs> --did-sep=<separator> --did-div=<divider> --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all meta data records:")
	fmt.Println("foxden meta ls")
	fmt.Println("\n# list specific meta-data record:")
	fmt.Println("foxden meta view <DID>")
	fmt.Println("\n# remove meta-data record:")
	fmt.Println("foxden meta rm 123xyz")
	fmt.Println("\n# add meta-data record with given schema, file and did attributes which create a did value:")
	fmt.Printf("foxden meta add <schema> <file.json> --did-attrs=%s --did-sep=%s --did-div=%s\n", attrs, sep, div)
	fmt.Println("\n# the same as above since it is default values")
	fmt.Println("foxden meta add <schema> <file.json>")
	fmt.Println("\n# the same as above but provide json output")
	fmt.Println("foxden meta add <schema> <file.json> --json")
}

// helper function to add meta data record
func metaAddRecord(args []string, attrs, sep, div string, jsonOutput bool) {
	if len(args) == 1 {
		fmt.Println("manual insertion is not implemented yet")
		metaUsage()
		os.Exit(1)
	}
	if len(args) != 3 {
		metaUsage()
		os.Exit(1)
	}
	// user must provide client meta add schema file.json
	schemaName := args[1]
	fname := args[2]

	// check if given fname is a file
	_, err := os.Stat(fname)
	exit(fmt.Sprintf("unable to check file stat, file %s", fname), err)
	file, err := os.Open(fname)
	exit(fmt.Sprintf("unable to open file %s", fname), err)
	defer file.Close()
	data, err := io.ReadAll(file)
	exit(fmt.Sprintf("unable to read file %s", fname), err)
	var record map[string]any
	err = json.Unmarshal(data, &record)
	exit(fmt.Sprintf("unable to unmarshal data, file %s", fname), err)

	// add proper did
	did, ok := record["did"]
	if !ok || did == "" {
		did = utils.CreateDID(record, attrs, sep, div)
		record["did"] = did
	}

	// we need to create /meta/file/upload call using URL form
	var mrec services.MetaRecord
	mrec.Schema = schemaName
	mrec.Record = record
	data, err = json.MarshalIndent(mrec, "", "  ")
	exit("unable to marshal data", err)
	rurl := fmt.Sprintf("%s", _srvConfig.Services.MetaDataURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	exit("unable to fetch data from meta-data service", err)
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
			fmt.Printf("ERROR: failed to add record to MetaData service\n%+v", response.String())
		}
	}
}

// helper function to delete meta-data record
func metaDeleteRecord(args []string, jsonOutput bool) {
	if len(args) != 2 {
		metaUsage()
		os.Exit(1)
	}
	mid := args[1]
	token, err := accessToken()
	exit("", err)
	rurl := fmt.Sprintf("%s/meta/%s", _srvConfig.Services.MetaDataURL, mid)
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
		fmt.Printf("SUCCESS: record %s was successfully removed\n", mid)
	} else {
		fmt.Printf("WARNING: record %s failed to be removed\n", mid)
	}

}

// helper funtion to list meta-data records
func metaListRecord(user, spec string, jsonOutput bool) {
	records, err := getMeta(user, spec)
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

// helper function to print meta data records in Json format
func metaJsonRecord(user, did string, jsonOutput bool) {
	query := "did:" + did
	log.Println("### query", query)
	records, err := getMeta(user, query)
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

func metaCommand() *cobra.Command {
	attrs, sep, div := didMetaData()
	cmd := &cobra.Command{
		Use:   "meta",
		Short: "foxden MetaData commands",
		Long:  "foxden MetaData commands to access FOXDEN MetaData service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			attrs, _ := cmd.Flags().GetString("did-attrs")
			sep, _ := cmd.Flags().GetString("did-sep")
			div, _ := cmd.Flags().GetString("did-div")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			if len(args) == 0 {
				metaUsage()
			} else if args[0] == "ls" {
				user, _ := getUserToken()
				if len(args) == 2 {
					metaListRecord(user, args[1], jsonOutput)
				} else {
					metaListRecord(user, "", jsonOutput)
				}
			} else if args[0] == "view" {
				user, _ := getUserToken()
				if len(args) == 2 {
					metaJsonRecord(user, args[1], jsonOutput)
				} else {
					metaJsonRecord(user, "", jsonOutput)
				}
			} else if args[0] == "add" {
				writeToken()
				metaAddRecord(args, attrs, sep, div, jsonOutput)
			} else if args[0] == "rm" {
				writeToken()
				metaDeleteRecord(args, jsonOutput)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().String("did-attrs", attrs, "did attributes")
	cmd.PersistentFlags().String("did-sep", sep, "did separator")
	cmd.PersistentFlags().String("did-div", div, "did key-value divider")
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		metaUsage()
		return nil
	})
	return cmd
}
