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
	"strings"
	"time"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

func metaRecords(user, query string, skeys []string, sorder int) ([]map[string]any, error) {
	var records []map[string]any
	rec := services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query, Idx: 0, Limit: -1, SortKeys: skeys, SortOrder: sorder},
	}
	data, err := json.Marshal(rec)
	rurl := fmt.Sprintf("%s/search", srvConfig.Config.Services.DiscoveryURL)
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
	fmt.Println("       search keys are case-incensitive")
	fmt.Println("options: --sort-key --sort-order --json --elapsed-time")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all known search keys:")
	fmt.Println("foxden search keys")
	fmt.Println("\n# search CHESS data using query language, e.g. empty query match all records")
	fmt.Println("foxden search {}")
	fmt.Println("\n# same as above but provide output in JSON data-format:")
	fmt.Println("foxden search {} --json")
	fmt.Println("\n# search using query language,")
	fmt.Println("# provide valid JSON use single quotes around it and double quotes for key:value pairs")
	fmt.Println("foxden search '{\"pi\":\"name\"}'")
	fmt.Println("\n# search using key:value pairs, e.g. pi:name where 'pi' is record key and 'name' would be PI user name")
	fmt.Println("# keys can be in lower case, e.g. pi instead of PI used in meta-data record")
	fmt.Println("foxden search pi:name")
	fmt.Println("\n# same as above but provide output in JSON data-format:")
	fmt.Println("foxden search pi:name --json")
	fmt.Println("\n# same as above but provide sorting order:")
	fmt.Println("foxden search pi:name --sort-keys=date --sort-order=1")
}

// helper function to get all known search (QL) keys across all FOXDEN services
func getSearchKeys() []string {
	urls := []string{
		srvConfig.Config.Services.DataBookkeepingURL,
		srvConfig.Config.Services.DataManagementURL,
		srvConfig.Config.Services.MetaDataURL,
		srvConfig.Config.Services.SpecScansURL,
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
func searchListRecord(user, spec string, skeys []string, sorder int, jsonOutput, elapsedTime bool) {
	defer TrackTime(elapsedTime)()
	if spec == "keys" {
		skeys := getSearchKeys()
		fmt.Println("FOXDEN search keys:")
		for _, k := range skeys {
			fmt.Println(k)
		}
		return
	}
	spec = utils.NormalizeSpec(spec)
	records, err := metaRecords(user, spec, skeys, sorder)
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
		fmt.Printf("did        : %v\n", did)
		fmt.Printf("schema     : %v\n", r["schema"])
		fmt.Printf("cycle      : %v\n", r["cycle"])
		fmt.Printf("beamline   : %v\n", r["beamline"])
		fmt.Printf("btr        : %v\n", r["btr"])
		fmt.Printf("sample_name: %v\n", r["sample_name"])
		tstamp := "Not Available"
		if val, ok := r["date"]; ok {
			secondsSinceEpoch := int64(val.(float64))
			tstamp = time.Unix(secondsSinceEpoch, 0).Format(time.RFC3339)
		}
		fmt.Printf("date       : %v\n", tstamp)
		//         fmt.Println(fmt.Sprintf("%f", val), int64(val.(float64)), did)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")
}

// helper function to print search data records in Json format
func searchJsonRecord(user, did string, skeys []string, sorder int) {
	query := "did:" + did
	records, err := metaRecords(user, query, skeys, sorder)
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
			jsonOutput, _ := cmd.Flags().GetBool("json")
			elapsedTime, _ := cmd.Flags().GetBool("elapsed-time")
			sortKeys, _ := cmd.Flags().GetString("sort-keys")
			sortOrder, _ := cmd.Flags().GetInt("sort-order")
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
			user, _ := getUserToken()
			if len(args) == 0 {
				searchUsage()
			} else {
				searchListRecord(user, args[0], skeys, sortOrder, jsonOutput, elapsedTime)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().Bool("elapsed-time", false, "print out elapsed time")
	cmd.PersistentFlags().String("sort-keys", "date", "sort key(s), if multiple keys separate them by comma (default: date)")
	cmd.PersistentFlags().Int("sort-order", -1, "sort order: 1 ascending, -1 desecnding (default)")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		searchUsage()
		return nil
	})
	return cmd
}
