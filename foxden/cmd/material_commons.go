package cmd

// CHESComputing foxden tool: material commons module
//
// Copyright (c) 2024 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// helper function to get metadata
// MaterialCommons represents MaterialCommons object returned from discovery service
type MaterialCommons struct {
	ID          string   `json:"id"`
	Site        string   `json:"site"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Bucket      string   `json:"bucket"`
}

// MetaDataRecord represents MetaData record returned by discovery service
type MCDataRecord struct {
	Status string            `json:"status"`
	Data   []MaterialCommons `json:"data"`
}

// MCResponse represents HTTP response from Material Commons API
type MCResponse struct {
	Data []map[string]any `json:"data"`
}

// helper function to get meta-data records
func getMaterialCommons(user, query string) ([]map[string]any, error) {
	var records []map[string]any
	materialCommonsUrl := "https://materialscommons.org/api"
	if _srvConfig.MaterialCommons.Url == "" {
		materialCommonsUrl = _srvConfig.MaterialCommons.Url
	}
	rurl := fmt.Sprintf("%s/projects", materialCommonsUrl)
	_httpReadRequest.Token = _srvConfig.MaterialCommons.Token
	//     resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("data:", string(data))
		exit("unable to read data from meta-data service", err)
	}
	var mcRecord MCResponse
	err = json.Unmarshal(data, &mcRecord)

	if err != nil {
		log.Println("response data", string(data))
		exit("Unable to unmarshal the data", err)
	}
	for _, rec := range mcRecord.Data {
		records = append(records, rec)
	}
	return records, nil
}

// helper function to provide usage of meta option
func mcUsage() {
	fmt.Println("foxden mc <ls|rm|view> [options]")
	fmt.Println("foxden mc add <file.json> {options}")
	fmt.Println("options: --schema=<schema> --did-attrs=<attrs> --did-sep=<separator> --did-div=<divider> --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all mc data records:")
	fmt.Println("foxden mc ls")
}

// helper funtion to list meta-data records
func mcListRecord(user, spec string, jsonOutput bool) {
	records, err := getMaterialCommons(user, spec)
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
		fmt.Printf("%+v", r)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")

}

func materialCommonsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mc",
		Short: "foxden MaterialCommons commands",
		Long:  "foxden MaterialCommons commands to access FOXDEN MaterialCommons service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
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
					mcListRecord(user, args[1], jsonOutput)
				} else {
					mcListRecord(user, "", jsonOutput)
				}
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		metaUsage()
		return nil
	})
	return cmd
}
