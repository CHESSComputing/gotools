package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

type DBSRecord map[string]any

// helper function to fetch data from DBS service
func getData(rurl string) []DBSRecord {
	var results []DBSRecord
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
	}
	if args[1] == "datasets" {
		rurl := fmt.Sprintf("%s/datasets", _srvConfig.Services.DataBookkeepingURL)
		for _, rec := range getData(rurl) {
			printResults(rec)
		}
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
	//     if len(args) != 1 {
	//         dbsUsage()
	//         os.Exit(1)
	//     }
	token, err := accessToken()
	checkError(err)
	// check if given args contains a file
	lastArg := args[len(args)-1]
	_, err = os.Stat(lastArg)
	checkError(err)
	file, err := os.Open(lastArg)
	checkError(err)
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	var rec ResponseRecord
	err = json.Unmarshal(data, &rec)
	checkError(err)

	rurl := fmt.Sprintf("%s/dataset", _srvConfig.Services.DataBookkeepingURL)
	req, err := http.NewRequest("POST", rurl, bytes.NewBuffer(data))
	checkError(err)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	checkError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	checkError(err)
	var response services.ServiceStatus
	err = json.Unmarshal(body, &response)
	checkError(err)
	if response.Status == "ok" {
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
		Short: "client dbs command",
		Long: `client data-bookkeeping system command
                Complete documentation is available at https://www.lepp.cornell.edu/CHESSComputing/documentation/`,
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				dbsUsage()
			} else if args[0] == "ls" {
				dbsListRecord(args)
			} else if args[0] == "add" {
				dbsAddRecord(args)
			} else if args[0] == "rm" {
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
