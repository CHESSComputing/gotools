package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/CHESSComputing/golib/mongo"
	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

func metaRecords(user, query string) ([]mongo.Record, error) {
	var records []mongo.Record
	rec := services.ServiceRequest{
		Client:       "client",
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
	records = response.Results.Records
	return records, nil
}

// helper function to provide usage of search option
func searchUsage() {
	fmt.Println("client search|view <spec>")
	fmt.Println("Examples:")
	fmt.Println("\n# search CHESS data:")
	fmt.Println("client search <spec>")
}

// helper funtion to list search-data records
func searchListRecord(user, spec string) {
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
		Short: "client search command",
		Long:  "client search-data command\n" + doc,
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
