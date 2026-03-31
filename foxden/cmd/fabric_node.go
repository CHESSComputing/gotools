package cmd

// CHESComputing foxden tool: s3 module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// IngestRecord represents ingest record from FabricNode
type IngestRecord struct {
	Ingested int    `json:"ingested"`
	Did      string `json:"did"`
	GraphIRI string `json:"graphIRI"`
}

// helper function to provide fabric usage info
func fabricUsage() {
	fmt.Println("foxden fabric <ls|ingest> [options]")
	fmt.Println("\nExamples:")
	fmt.Println("\n# ingest did into FabricNode:")
	fmt.Println("foxden fabric ingest <did>")
}

// helper function to list content of a bucket on s3 storage
func fabricList(args []string, jsonOutput bool) {
	// args contains [ls bucket]
	if args[0] != "ls" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}

	// get beamlines datasets from fabric node
	bl := args[1]
	rurl := fmt.Sprintf("%s/catalog/beamlines/%s/datasets",
		srvConfig.Config.Services.FabricCatalogURL, bl)

	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	if jsonOutput {
		if val, err := json.MarshalIndent(data, "", "  "); err == nil {
			fmt.Println(string(val))
		}
		return
	}
	printMap(data)
}

// helper function to ingest did into fabric node
func fabricIngest(args []string) {
	// args contains [create bucket]
	if len(args) != 2 {
		fmt.Println("ERROR: wrong number of arguments")
		os.Exit(1)
	}
	if args[0] != "ingest" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	did := args[1]
	bl := utils.GetBeamline(did)
	encodedDid := url.QueryEscape(did)
	rurl := fmt.Sprintf("%s/beamlines/%s/datasets/%s/foxden/ingest",
		srvConfig.Config.Services.FabricDataServiceURL, bl, encodedDid)
	if verbose > 0 {
		fmt.Println("HTTP POST", rurl)
	}
	resp, err := _httpWriteRequest.Post(rurl, "", bytes.NewBuffer([]byte{}))
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var results IngestRecord
	if err := dec.Decode(&results); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	fmt.Printf("fabric ingest results:\n%+v\n", results)
}

func fabricCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fabric",
		Short: "foxden fabric commands",
		Long:  "foxden fabric commands to access CHESS FabricNode service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if len(args) == 0 {
				fabricUsage()
			} else if args[0] == "ls" {
				accessToken()
				fabricList(args, jsonOutput)
			} else if args[0] == "ingest" {
				writeToken()
				fabricIngest(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		fabricUsage()
		return nil
	})
	return cmd
}
