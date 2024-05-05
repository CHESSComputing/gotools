package cmd

// CHESComputing foxden tool: search module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

func metaRecords(user, query string) ([]map[string]any, error) {
	var records []map[string]any
	rec := services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query, Idx: 0, Limit: -1},
	}
	data, err := json.Marshal(rec)
	rurl := fmt.Sprintf("%s/search", _srvConfig.Services.DiscoveryURL)
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("unable to fetch data from search-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from search-data service", err)
	}

	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		exit("Unable to unmarshal the data", err)
	}
	if response.HttpCode >= 400 {
		fmt.Printf("Service %s returned error: %v\n", response.Service, response.Error)
		os.Exit(1)
	}
	records = response.Results.Records
	return records, nil
}

// helper function to provide usage of search option
func searchUsage() {
	fmt.Println("foxden search <spec>")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all known search keys:")
	fmt.Println("foxden search keys")
	fmt.Println("\n# search CHESS data using query language, e.g. empty query match all records")
	fmt.Println("foxden search {}")
	fmt.Println("\n# search using query language,")
	fmt.Println("# provide valid JSON use single quotes around it and double quotes for key:value pairs")
	fmt.Println("foxden search '{\"PI\":\"name\"}'")
	fmt.Println("\n# search using key:value pairs, e.g. pi:name where 'pi' is record key and 'name' would be PI user name")
	fmt.Println("# keys can be in lower case, e.g. pi instead of PI used in meta-data record")
	fmt.Println("foxden search pi:name")
}

// helper function to get all known search (QL) keys across all FOXDEN services
func getSearchKeys() []string {
	urls := []string{
		_srvConfig.Services.DataBookkeepingURL,
		_srvConfig.Services.DataManagementURL,
		_srvConfig.Services.MetaDataURL,
		_srvConfig.Services.SpecScansURL,
	}
	var skeys []string
	for _, url := range urls {
		rurl := fmt.Sprintf("%s/qlkeys", url)
		resp, err := _httpReadRequest.Get(rurl)
		if err != nil {
			exit(fmt.Sprintf("unable to reach %s", rurl), err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			exit("unable to read HTTP response", err)
		}
		var records []string
		err = json.Unmarshal(data, &records)
		if err != nil {
			exit("unable to unmarshal HTTP response", err)
		}
		for _, k := range records {
			skeys = append(skeys, k)
		}
	}
	skeys = utils.List2Set(skeys)
	sort.Strings(skeys)
	return skeys
}

// helper function to list search-data records
func searchListRecord(user, spec string) {
	if spec == "keys" {
		skeys := getSearchKeys()
		fmt.Println("FOXDEN search keys:")
		for _, k := range skeys {
			fmt.Println(k)
		}
		return
	}
	records, err := metaRecords(user, spec)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	for _, r := range records {
		fmt.Println("---")
		val := r["did"]
		var did string
		switch vvv := val.(type) {
		case float64:
			did = fmt.Sprintf("%d", int64(val.(float64)))
		case int64, int32, int:
			did = fmt.Sprintf("%d", int64(val.(int64)))
		default:
			did = vvv.(string)
		}
		fmt.Printf("DID     : %v\n", did)
		fmt.Printf("Schema  : %v\n", r["Schema"])
		fmt.Printf("Cycle   : %v\n", r["Cycle"])
		fmt.Printf("Beamline: %v\n", r["Beamline"])
		fmt.Printf("BTR     : %v\n", r["BTR"])
		fmt.Printf("Sample  : %v\n", r["Sample"])
		//         fmt.Println(fmt.Sprintf("%f", val), int64(val.(float64)), did)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")
}

// helper function to print search data records in Json format
func searchJsonRecord(user, did string) {
	query := "did:" + did
	records, err := metaRecords(user, query)
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

func searchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "foxden search commands",
		Long:  "foxden search commands to access FOXDEN DataDiscovery service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			user, _ := getUserToken()
			if len(args) == 0 {
				searchUsage()
			} else {
				searchListRecord(user, args[0])
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		searchUsage()
		return nil
	})
	return cmd
}
