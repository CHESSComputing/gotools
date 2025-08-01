package cmd

// CHESComputing foxden tool: view module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/spf13/cobra"
)

// helper function to provide usage of view option
func viewUsage() {
	fmt.Println("foxden view <DID> [options]")
	fmt.Println("options: --parents --children --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# view details of specific did record:")
	fmt.Println("foxden view /beamline=test/btr=test-123-a/cycle=2023-3/sample_name=sample")
	fmt.Println("\n# view details of specific did record including parents and children:")
	fmt.Println("foxden view /beamline=test/btr=test-123-a/cycle=2023-3/sample_name=sample --parents --children")
}

func viewRecord(user, did string, parents, children, jsonOutput bool) {
	records, provRecords := getRecords(user, did, parents, children)
	metadata, err1 := json.MarshalIndent(records, "", "  ")
	provdata, err2 := json.MarshalIndent(provRecords, "", "  ")
	if err1 != nil {
		exit("unable to serialize metadata records", err1)
	}
	if err2 != nil {
		exit("unable to serialize metadata records", err2)
	}
	fmt.Println("\n### Metadata records\n")
	print(string(metadata))
	fmt.Println("\n\n### Provenance records\n")
	print(string(provdata))
}

func getRecords(user, did string, parents, children bool) ([]map[string]any, []map[string]any) {
	// get meta-data records
	query := "did:" + did
	var skeys []string
	sorder := 0
	records, err := metaRecords(user, query, skeys, sorder)
	if err != nil {
		exit("unable to marshal data", err)
	}

	// get provenance records
	provRecords := getProvRecords(did, "provenance")
	if parents {
		for _, r := range getRecursiveRecords(did, "parents", "parent_id", nil) {
			provRecords = append(provRecords, r)
		}
	}
	if children {
		for _, r := range getRecursiveRecords(did, "children", "child_id", nil) {
			provRecords = append(provRecords, r)
		}
	}
	return records, provRecords
}

func getRecursiveRecords(did string, api, key string, seen map[string]bool) []map[string]any {
	if seen == nil {
		seen = make(map[string]bool)
	}
	if seen[did] {
		return nil
	}
	seen[did] = true

	var allParents []map[string]any
	records := getProvRecords(did, api)
	for _, r := range records {
		allParents = append(allParents, r)
		if val, ok := r[key]; ok {
			pdid, ok := val.(string)
			if ok {
				subParents := getRecursiveRecords(pdid, api, key, seen)
				allParents = append(allParents, subParents...)
			}
		}
	}
	return allParents
}

func getProvRecords(did, api string) []map[string]any {
	rurl := fmt.Sprintf("%s/%s?did=%s", srvConfig.Config.Services.DataBookkeepingURL, api, did)
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit(fmt.Sprintf("unable to fetch data from provenance service API %s", api), err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from provenance service", err)
	}
	var provRecords []map[string]any
	err = json.Unmarshal(data, &provRecords)
	if err != nil {
		exit("unable to read data from provenance service", err)
	}
	return provRecords
}

func printJsonRecords(records []map[string]any, jsonOutput bool) {
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

// helper function to print view data records in Json format
func viewMetaRecord(user, did string, jsonOutput bool) {
	query := "did:" + did
	var skeys []string
	sorder := 0
	records, err := metaRecords(user, query, skeys, sorder)
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
		fmt.Println("### MetaData records:")
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			exit("unable to marshal data", err)
		}
		fmt.Println(string(data))
	}
}

// helper function to look-up DBS records
func viewDBSRecord(user, did string, parents, children, jsonOutput bool) {
	// look-up dataset records
	rurl := fmt.Sprintf("%s/datasets?did=%s", srvConfig.Config.Services.DataBookkeepingURL, did)
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from search-data service", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from search-data service", err)
	}
	fmt.Println("### Provenance dataset records:")
	fmt.Println(string(data))

	// look-up files records
	rurl = fmt.Sprintf("%s/files?did=%s", srvConfig.Config.Services.DataBookkeepingURL, did)
	resp, err = _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from search-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from search-data service", err)
	}
	fmt.Println("### Provenance files records:")
	fmt.Println(string(data))

	// look-up parents info
	rurl = fmt.Sprintf("%s/parents?did=%s", srvConfig.Config.Services.DataBookkeepingURL, did)
	resp, err = _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from search-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from search-data service", err)
	}
	fmt.Println("### Provenance parents records:")
	fmt.Println(string(data))
}

func viewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "foxden view commands",
		Long:  "foxden view data-record commands via FOXDEN services\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			parents, _ := cmd.Flags().GetBool("parents")
			children, _ := cmd.Flags().GetBool("children")
			if len(args) == 0 {
				viewUsage()
			} else {
				user, _ := getUserToken()
				did := args[0]
				viewRecord(user, did, parents, children, jsonOutput)
			}
		},
	}
	cmd.PersistentFlags().Bool("parents", false, "recurse look-up of all parents")
	cmd.PersistentFlags().Bool("children", false, "recurse look-up of all children")
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		viewUsage()
		return nil
	})
	return cmd
}
