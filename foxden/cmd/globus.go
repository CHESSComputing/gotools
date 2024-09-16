package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	globus "github.com/CHESSComputing/golib/globus"
	"github.com/spf13/cobra"
)

// helper function to provide usage of meta option
func globusUsage() {
	fmt.Println("foxden globus <ls|rm|view> [options]")
	fmt.Println("foxden globus add <file.json> {options}")
	fmt.Println("options: --scope=<globus_scopes> --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# search Globus records within CHESS pattern:")
	fmt.Println("foxden globus search CHESS")
	fmt.Println("\n# list all globus data records:")
	fmt.Println("foxden globus ls")
}

// helper function to get meta-data records
func getGlobus(token, query string) ([]map[string]any, error) {
	var records []map[string]any
	globus.Search(token, query)
	return records, nil
}

// helper function to list meta-data records
func globusListRecord(token, spec string, jsonOutput bool) {
	records, err := getGlobus(token, spec)
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

func globusSearch(token, pat string, jsonOutput bool) {
	records := globus.Search(token, pat)
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
		fmt.Printf("ID:          %s\n", r.Id)
		fmt.Printf("Name:        %s\n", r.Name)
		fmt.Printf("Owner:       %s\n", r.Owner)
		fmt.Printf("Description: %s\n", r.Description)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")
}

func globusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "globus",
		Short: "foxden Globus commands",
		Long:  "foxden Globus commands to access Globus services through FOXDEN\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			verbose, _ := cmd.Flags().GetInt("verbose")
			if verbose > 0 {
				globus.Verbose = verbose
			}
			scope := "urn:globus:auth:scope:transfer.api.globus.org:all"
			token, err := globus.Token(scope)
			if err != nil {
				log.Println("ERROR", err)
			}
			if len(args) == 0 {
				metaUsage()
			} else if args[0] == "ls" {
				if len(args) == 2 {
					globusListRecord(token, args[1], jsonOutput)
				} else {
					globusListRecord(token, "", jsonOutput)
				}
			} else if args[0] == "search" {
				pat := ""
				if len(args) == 2 {
					pat = args[1]
				}
				globusSearch(token, pat, jsonOutput)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().Int("verbose", 0, "verbosity level")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		metaUsage()
		return nil
	})
	return cmd
}
