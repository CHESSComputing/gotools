package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// helper function to provide data usage info
func dmUsage() {
	fmt.Println("foxden data <files> [options]")
	fmt.Println("options:")
	fmt.Println("         --did=<did>")
	fmt.Println("         --ext=<ext>")
	fmt.Println("\nExamples:")
	fmt.Println("\n# get data snapshot for given did:")
	fmt.Println("foxden data --did=$did")
	fmt.Println("\n# get data files for given did and extension:")
	fmt.Println("foxden data files --did=$did --ext=<ext>")
	fmt.Println()
}

// DMRecord represents DataManagement record
type DMRecord struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// helper function to get dm data for given did
func dmData(did string) {
	// make HTTP call to DataManagement
	rurl := fmt.Sprintf("%s/data?did=%s", _srvConfig.DataManagementURL, did)
	// Create a new HTTP request to the target URL
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to query DataManagement service", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data", err)
		return
	}
	var records []DMRecord
	err = json.Unmarshal(data, &records)
	if err != nil {
		exit("unable to unmarshal data", err)
	}
	for _, rec := range records {
		if rec.IsDir {
			fmt.Println("Dir :", rec.Name)
		} else {
			fmt.Println("File:", rec.Name)
		}
	}
}

// helper function to get list of files for given did and file extension
func dmFiles(did, ext string) {
	// make HTTP call to DataManagement
	pat := fmt.Sprintf("(?i).*%s$", ext)
	if ext == "" || ext == "all" {
		pat = "all"
	}
	rurl := fmt.Sprintf("%s/files?did=%s&pattern=%s", _srvConfig.DataManagementURL, did, pat)

	// Create a new HTTP request to the target URL
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to query DataManagement service", err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data", err)
		return
	}
	var files []string
	err = json.Unmarshal(data, &files)
	if err != nil {
		exit("unable to unmarshal the data", err)
		return
	}
	for _, f := range files {
		fmt.Println(f)
	}
}

func dmCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "foxden data command",
		Long:  "foxden data command to access FOXDEN Publication service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			ext, _ := cmd.Flags().GetString("ext")
			did, _ := cmd.Flags().GetString("did")
			accessToken()
			if len(args) == 0 {
				dmData(did)
			} else if args[0] == "files" {
				dmFiles(did, ext)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.PersistentFlags().String("did", "", "did string")
	cmd.PersistentFlags().String("ext", "", "ext string")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		dmUsage()
		return nil
	})
	return cmd
}
