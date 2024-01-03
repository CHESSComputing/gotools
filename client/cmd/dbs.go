package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	dbs "github.com/CHESSComputing/DataBookkeeping/dbs"
	"github.com/spf13/cobra"
)

type DBSRecord map[string]any

// helper function to fetch data from DBS service
func getData(rurl string) []DBSRecord {
	var results []DBSRecord
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}
	resp, err := _httpReadRequest.Get(rurl)
	//     resp, err := http.Get(rurl)
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

// helper function to print dbs record items
func printResults(rec DBSRecord) {
	fmt.Println("---")
	maxKey := 0
	for key, _ := range rec {
		if len(key) > maxKey {
			maxKey = len(key)
		}
	}
	for key, val := range rec {
		pad := strings.Repeat(" ", maxKey-len(key))
		fmt.Printf("%s%s\t%v\n", key, pad, val)
	}
}

// helper function to list dataset information
func dbsListRecord(args []string) {
	if len(args) == 1 {
		fmt.Println("WARNING: please provide dbs attribute")
		os.Exit(1)
	} else if args[1] == "datasets" {
		rurl := fmt.Sprintf("%s/datasets", _srvConfig.Services.DataBookkeepingURL)
		for _, rec := range getData(rurl) {
			printResults(rec)
		}
		//     } else if args[1] == "files" {
		//         rurl := fmt.Sprintf("%s/files", _srvConfig.Services.DataBookkeepingURL)
		//         for _, rec := range getData(rurl) {
		//             printResults(rec)
		//         }
		//     } else if args[1] == "buckets" {
		//         rurl := fmt.Sprintf("%s/buckets", _srvConfig.Services.DataBookkeepingURL)
		//         for _, rec := range getData(rurl) {
		//             printResults(rec)
		//         }
	} else {
		fmt.Println("Not implemented yet")
	}
}

// ResponseRecord represents MetaData record returned by discovery service
type ResponseRecord struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// helper function to add dataset information
func dbsAddRecord(args []string) {
	// check if given args contains a file
	lastArg := args[len(args)-1]
	_, err := os.Stat(lastArg)
	exit("", err)
	file, err := os.Open(lastArg)
	exit("", err)
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	var rec dbs.DatasetRecord
	err = json.Unmarshal(data, &rec)
	exit("", err)

	rurl := fmt.Sprintf("%s/dataset", _srvConfig.Services.DataBookkeepingURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	//     defer resp.Body.Close()
	//     body, err := io.ReadAll(resp.Body)
	//     exit("", err)
	//     fmt.Printf("#### dbs returni body='%s' response %+v", string(body), resp)
	//     var response services.ServiceResponse
	//     err = json.Unmarshal(body, &response)
	//     exit("", err)
	//     if response.Status == "ok" {
	if resp.StatusCode == 200 {
		fmt.Printf("SUCCESS: dbs record was successfully added\n")
	} else {
		fmt.Printf("WARNING: dbs record failed to be added dbs service\n")
	}
}

// helper function to delete dataset information
func dbsDeleteRecord(args []string) {
}

// helper function to provide usage of dbs option
func dbsUsage() {
	fmt.Println("client dbs <ls|add|rm> [value]")
	fmt.Println("Examples:")
	fmt.Println("\n# list all dbs records:")
	fmt.Println("client dbs ls <dataset|site|file>")
	fmt.Println("\n# remove dbs-data record:")
	fmt.Println("client dbs rm <dataset|site|file>")
	fmt.Println("\n# add dbs-data record:")
	fmt.Println("client dbs add <dataset|site|file>")
}
func dbsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dbs",
		Short: "client provenance (dbs) commands",
		Long:  "client provenance data-bookkeeping system (dbs) commands\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				dbsUsage()
			} else if args[0] == "ls" {
				// obtain valid access token
				accessToken()
				dbsListRecord(args)
			} else if args[0] == "add" {
				writeToken()
				dbsAddRecord(args)
			} else if args[0] == "rm" {
				writeToken()
				dbsDeleteRecord(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		dbsUsage()
		return nil
	})
	return cmd
}
