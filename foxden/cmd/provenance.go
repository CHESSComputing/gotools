package cmd

// CHESComputing foxden tool: dbs module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
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
func provListRecord(args []string, dataset, dfile string) {
	var rurl string
	if len(args) == 1 {
		fmt.Println("WARNING: please provide provenance attribute")
		os.Exit(1)
	} else if args[1] == "datasets" {
		rurl = fmt.Sprintf("%s/datasets", _srvConfig.Services.DataBookkeepingURL)
		if dataset != "" {
			rurl = fmt.Sprintf("%s?dataset=%s", rurl, dataset)
		} else if dfile != "" {
			rurl = fmt.Sprintf("%s?file=%s", rurl, dfile)
		}
	} else if args[1] == "files" {
		rurl = fmt.Sprintf("%s/files", _srvConfig.Services.DataBookkeepingURL)
		if dataset != "" {
			rurl = fmt.Sprintf("%s?dataset=%s", rurl, dataset)
		} else if dfile != "" {
			rurl = fmt.Sprintf("%s?file=%s", rurl, dfile)
		}
	} else if args[1] == "buckets" {
		rurl = fmt.Sprintf("%s/buckets", _srvConfig.Services.DataBookkeepingURL)
	} else {
		fmt.Println("Not implemented yet")
		return
	}
	for _, rec := range getData(rurl) {
		printResults(rec)
	}
}

// ResponseRecord represents MetaData record returned by discovery service
type ResponseRecord struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// helper function to add dataset information
func provAddRecord(args []string) {
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
		fmt.Printf("SUCCESS: provenance record was successfully added\n")
	} else {
		fmt.Printf("WARNING: fail to add provenance record\n")
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		var records []map[string]any
		err = json.Unmarshal(data, &records)
		if err == nil {
			for _, rec := range records {
				fmt.Println(rec)
			}
		} else {
			fmt.Printf("HTTP response: %+v, error %v\n", string(data), err)
		}
	}
}

// helper function to delete dataset information
func provDeleteRecord(args []string) {
}

// helper function to provide usage of dbs option
func provUsage() {
	fmt.Println("foxden prov <ls|add> [options]")
	fmt.Println("options: provenance attributes like dataset(s), file(s)")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all provenance records:")
	fmt.Println("foxden prov ls <datasets|files>")
	//     fmt.Println("\n# remove provenance data record:")
	//     fmt.Println("foxden prov rm <dataset|site|file>")
	fmt.Println("\n# add provenance data record:")
	fmt.Println("foxden prov add <provenance.json>")
}
func provCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prov",
		Short: "foxden provenance commands",
		Long:  "foxden provenance commands to access FOXDEN Provenance service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			file, _ := cmd.Flags().GetString("file")
			dset, _ := cmd.Flags().GetString("dataset")
			if len(args) == 0 {
				provUsage()
			} else if args[0] == "ls" {
				// obtain valid access token
				accessToken()
				provListRecord(args, dset, file)
			} else if args[0] == "add" {
				writeToken()
				provAddRecord(args)
				//             } else if args[0] == "rm" {
				//                 deleteToken()
				//                 provDeleteRecord(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().String("dataset", "", "dataset to use")
	cmd.PersistentFlags().String("file", "", "file to use")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		provUsage()
		return nil
	})
	return cmd
}
