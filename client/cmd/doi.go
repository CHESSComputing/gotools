package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// helper function to provide doi usage info
func doiUsage() {
	fmt.Println("client doi <ls|create|add> [values]")
	fmt.Println("Examples:")
	fmt.Println("\n# create new doi document:")
	fmt.Println("client doi create")
	fmt.Println("\n# upload new document to existing doi doc:")
	fmt.Println("client doi add /path/file.md")
	fmt.Println("\n# list existing documents:")
	fmt.Println("client doi ls")
}

// helper function to list existing documents
func doiDocs(args []string) {
	rurl := fmt.Sprintf("%s/docs", _srvConfig.Services.PublicationURL)
	if len(args) == 2 {
		rurl += fmt.Sprintf("/%s", args[1])
	}
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if len(args) == 2 {
		var rec map[string]any
		err = json.Unmarshal(data, &rec)
		printMap(rec)
		return
	}
	var records []map[string]any
	err = json.Unmarshal(data, &records)
	for _, rec := range records {
		printMap(rec)
	}
}

// helper function to create new bucket on doi storage
func doiCreate(args []string) {
}

// helper function to upload file or directory to bucket on doi storage
func doiAdd(args []string) {
}

func doiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doi",
		Short: "client doi command",
		Long:  "client doi command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				doiUsage()
			} else if args[0] == "ls" {
				accessToken()
				doiDocs(args)
			} else if args[0] == "create" {
				writeToken()
				doiCreate(args)
			} else if args[0] == "add" {
				writeToken()
				doiAdd(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		doiUsage()
		return nil
	})
	return cmd
}
