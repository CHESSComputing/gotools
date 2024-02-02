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

	services "github.com/CHESSComputing/golib/services"
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

	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		log.Println("response data", string(data))
		exit("Unable to unmarshal the data", err)
	}
	records = response.Results.Records
	return records, nil
}

// helper function to provide usage of meta option
func metaUsage() {
	fmt.Println("foxden meta <ls|add|rm> [value]")
	fmt.Println("Examples:")
	fmt.Println("\n# list all meta data records:")
	fmt.Println("foxden meta ls")
	fmt.Println("\n# list specific meta-data record:")
	fmt.Println("foxden meta view <DID>")
	fmt.Println("\n# remove meta-data record:")
	fmt.Println("foxden meta rm 123xyz")
	fmt.Println("\n# add meta-data record:")
	fmt.Println("foxden meta add <schema> <file.json>")
}

// helper function to add meta data record
func metaAddRecord(args []string) {
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
	exit("unable to check file stat", err)
	file, err := os.Open(fname)
	exit("unable to open file", err)
	defer file.Close()
	data, err := io.ReadAll(file)
	exit("unable to read file", err)
	var record map[string]any
	err = json.Unmarshal(data, &record)
	exit("unable to unmarshal data", err)

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

	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)
	exit("Unable to unmarshal the data", err)
	if response.Status == "ok" {
		fmt.Printf("SUCCESS: record was successfully added\n")
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
func metaDeleteRecord(args []string) {
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
	if response.Status == "ok" {
		fmt.Printf("SUCCESS: record %s was successfully removed\n", mid)
	} else {
		fmt.Printf("WARNING: record %s failed to be removed\n", mid)
	}

}

// helper funtion to list meta-data records
func metaListRecord(user, spec string) {
	records, err := getMeta(user, spec)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	for _, r := range records {
		fmt.Println("---")
		val := r["did"]
		did := fmt.Sprintf("%d", int64(val.(float64)))
		fmt.Printf("DID     : %v\n", did)
		fmt.Printf("Schema  : %v\n", r["Schema"])
		fmt.Printf("Cycle   : %v\n", r["Cycle"])
		fmt.Printf("Beamline: %v\n", r["Beamline"])
		fmt.Printf("BTR     : %v\n", r["BTR"])
		fmt.Printf("Sample  : %v\n", r["Sample"])
		//         fmt.Printf("%+v", r)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")

}

// helper function to print meta data records in Json format
func metaJsonRecord(user, did string) {
	query := "did:" + did
	records, err := getMeta(user, query)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
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
	cmd := &cobra.Command{
		Use:   "meta",
		Short: "foxden MetaData commands",
		Long:  "foxden MetaData commands\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				metaUsage()
			} else if args[0] == "ls" {
				user, _ := getUserToken()
				if len(args) == 2 {
					metaListRecord(user, args[1])
				} else {
					metaListRecord(user, "")
				}
			} else if args[0] == "view" {
				user, _ := getUserToken()
				if len(args) == 2 {
					metaJsonRecord(user, args[1])
				} else {
					metaJsonRecord(user, "")
				}
			} else if args[0] == "add" {
				writeToken()
				metaAddRecord(args)
			} else if args[0] == "rm" {
				writeToken()
				metaDeleteRecord(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		metaUsage()
		return nil
	})
	return cmd
}
