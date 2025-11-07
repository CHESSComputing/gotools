package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	globus "github.com/CHESSComputing/golib/globus"
	"github.com/spf13/cobra"
)

// helper function to provide usage of globus option
func globusUsage() {
	fmt.Println("foxden globus <ls|search|link> [options]")
	fmt.Println("options: --scope=<globus_scopes> --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# search Globus records within CHESS pattern:")
	fmt.Println("foxden globus search CHESS")
	fmt.Println("\n# list all globus data records:")
	fmt.Println("foxden globus ls <id:/path>")
	fmt.Println("\n# create globus data link:")
	fmt.Println("foxden globus link </path>")
}

// helper function to get globus-data records
func getGlobus(token, query string) ([]map[string]any, error) {
	var records []map[string]any
	globus.Search(token, query)
	return records, nil
}

// helper function to list globus-data records
func globusListRecord(token, spec string, jsonOutput bool) {
	arr := strings.Split(spec, ":")
	if len(arr) > 2 {
		exit(fmt.Sprintf("unable to extract endpoint id and path from the spec %s", spec), errors.New("insufficient argument"))
	}
	endpointId := arr[0]
	path := ""
	if len(arr) == 2 {
		path = arr[1]
	}
	globus.Ls(token, endpointId, path)
	/*
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
	*/

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

func globusLink(pat, path string) {
	gurl, err := globus.ChessGlobusLink(pat, path)
	exit("ChessGlobusLink", err)
	fmt.Println(gurl)
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
			scopes := []string{"urn:globus:auth:scope:transfer.api.globus.org:all"}
			if len(args) == 0 {
				globusUsage()
			} else if args[0] == "ls" {
				token, err := globus.Token(scopes)
				if err != nil {
					log.Fatalf("ERROR: unable to get globus token with scopes=%v, error=%v", scopes, err)
				}
				if len(args) == 2 {
					eid := strings.Split(args[1], ":")[0]
					globusListRecord(token, eid, jsonOutput)
				} else {
					globusListRecord(token, "", jsonOutput)
				}
			} else if args[0] == "search" {
				pat := ""
				if len(args) == 2 {
					pat = args[1]
				}
				token, err := globus.Token(scopes)
				if err != nil {
					log.Fatalf("ERROR: unable to get globus token with scopes=%v, error=%v", scopes, err)
				}
				globusSearch(token, pat, jsonOutput)
			} else if args[0] == "link" {
				pat := "CHESS Raw"
				path := ""
				if len(args) == 2 {
					path = args[1]
				}
				globusLink(pat, path)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().Int("verbose", 0, "verbosity level")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		globusUsage()
		return nil
	})
	return cmd
}
