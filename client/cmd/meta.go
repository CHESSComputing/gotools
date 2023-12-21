package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/CHESSComputing/golib/mongo"
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

func getMeta(query string) ([]mongo.Record, error) {
	var records []mongo.Record
	rec := make(map[string]string)
	rec["query"] = query
	rec["user"] = "cli"
	rec["client"] = "cli"
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
		exit("Unable to unmarshal the data", err)
	}
	records = response.Results.Records
	return records, nil
}

// helper function to provide usage of meta option
func metaUsage() {
	fmt.Println("client meta <ls|add|rm> [value]")
	fmt.Println("Examples:")
	fmt.Println("\n# list all meta data records:")
	fmt.Println("client meta ls")
	fmt.Println("\n# list specific meta-data record:")
	fmt.Println("client meta view <DID>")
	fmt.Println("\n# remove meta-data record:")
	fmt.Println("client meta rm 123xyz")
	fmt.Println("\n# add meta-data record:")
	fmt.Println("client meta add")
}

// helper function to add meta data record
func metaAddRecord(args []string) {
	if len(args) != 1 {
		metaUsage()
		os.Exit(1)
	}
	/*
		token, err := accessToken()
		exit("", err)
		site := inputPrompt("Site name:")
		description := inputPrompt("Site description:")
		bucket := inputPrompt("Site bucket:")
		var tags []string
		for _, r := range strings.Split(inputPrompt("Site tags (command separated):"), ",") {
			tags = append(tags, strings.Trim(r, " "))
		}
		meta := MetaData{
			Site:        site,
			Description: description,
			Bucket:      bucket,
			Tags:        tags,
		}
		data, err := json.Marshal(meta)
		exit("", err)
		rurl := fmt.Sprintf("%s/meta", _srvConfig.Services.MetaDataURL)
		req, err := http.NewRequest("POST", rurl, bytes.NewBuffer(data))
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
			fmt.Printf("SUCCESS: record %+v was successfully added\n", meta)
		} else {
			fmt.Printf("WARNING: record %+v failed to be added MetaData service\n", meta)
		}
	*/
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
func metaListRecord(spec string) {
	records, err := getMeta(spec)
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
		fmt.Println(fmt.Sprintf("%f", val), int64(val.(float64)), did)
	}
}

// helper function to print meta data records in Json format
func metaJsonRecord(did string) {
	query := "did:" + did
	records, err := getMeta(query)
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
		Short: "client meta command",
		Long:  "client meta-data command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				metaUsage()
			} else if args[0] == "ls" {
				accessToken()
				if len(args) == 2 {
					metaListRecord(args[1])
				} else {
					metaListRecord("")
				}
			} else if args[0] == "view" {
				accessToken()
				if len(args) == 2 {
					metaJsonRecord(args[1])
				} else {
					metaJsonRecord("")
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
