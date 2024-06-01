package cmd

// CHESComputing foxden tool: dbs module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	dbs "github.com/CHESSComputing/DataBookkeeping/dbs"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// helper function to fetch data from DBS service
func getData(rurl string) []MapRecord {
	var results []MapRecord
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}
	resp, err := _httpReadRequest.Get(rurl)
	//     resp, err := http.Get(rurl)
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

/*
// helper function to print dbs record items
func printRecord(rec MapRecord) {
	fmt.Println("---")
	maxKey := 0
	for key, _ := range rec {
		if len(key) > maxKey {
			maxKey = len(key)
		}
	}
	keys := utils.MapKeys(rec)
	sort.Strings(keys)
	for _, key := range keys {
		val, _ := rec[key]
		pad := strings.Repeat(" ", maxKey-len(key))
		fmt.Printf("%s%s\t%v\n", key, pad, val)
	}
}
*/

// helper function to list dataset information
func provListRecord(args []string, did, dfile string) {
	var rurl string
	if len(args) == 1 {
		fmt.Println("WARNING: please provide provenance attribute")
		os.Exit(1)
	} else if args[1] == "datasets" {
		rurl = fmt.Sprintf("%s/datasets", _srvConfig.Services.DataBookkeepingURL)
		if did != "" {
			rurl = fmt.Sprintf("%s?did=%s", rurl, did)
		} else if dfile != "" {
			rurl = fmt.Sprintf("%s?file=%s", rurl, dfile)
		}
	} else if args[1] == "files" {
		rurl = fmt.Sprintf("%s/files", _srvConfig.Services.DataBookkeepingURL)
		if did != "" {
			rurl = fmt.Sprintf("%s?did=%s", rurl, did)
		} else if dfile != "" {
			rurl = fmt.Sprintf("%s?file=%s", rurl, dfile)
		}
	} else if args[1] == "parents" {
		rurl = fmt.Sprintf("%s/parents?did=%s", _srvConfig.Services.DataBookkeepingURL, did)
	} else if args[1] == "child" {
		rurl = fmt.Sprintf("%s/child?did=%s", _srvConfig.Services.DataBookkeepingURL, did)
	} else if args[1] == "buckets" {
		rurl = fmt.Sprintf("%s/buckets", _srvConfig.Services.DataBookkeepingURL)
	} else {
		fmt.Println("Not implemented yet")
		return
	}
	for _, rec := range getData(rurl) {
		// convert seconds since epoch to human readable string
		if v, ok := rec["create_at"]; ok {
			rec["create_at"] = parseTimestamp(fmt.Sprintf("%v", v))
		}
		if v, ok := rec["modify_at"]; ok {
			rec["modify_at"] = parseTimestamp(fmt.Sprintf("%v", v))
		}
		printRecord(rec, "---")
	}
}

func parseTimestamp(v string) string {
	ts, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Fatal("unable to parse input timestamp", v, " error: ", err)
	}
	tstmp := time.Unix(int64(ts), 0)
	return tstmp.String()
}

// ResponseRecord represents MetaData record returned by discovery service
type ResponseRecord struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// helper function to read input record
func readInput(args []string) ([]byte, error) {
	// check if given args contains a file
	lastArg := args[len(args)-1]
	_, err := os.Stat(lastArg)
	exit("", err)
	file, err := os.Open(lastArg)
	exit("", err)
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	return data, err
}

// helper function to print HTTP response
func printResponse(resp *http.Response, err error) {
	if err == nil && resp.StatusCode == 200 {
		fmt.Printf("SUCCESS: provenance record was successfully added\n")
	} else {
		if err != nil {
			fmt.Printf("ERROR: fail to add provenance record, error: %v\n", err)
		} else {
			fmt.Printf("WARNING: fail to add provenance record\n\n")
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			var records []map[string]any
			err = json.Unmarshal(data, &records)
			if err == nil {
				keys := []string{"code", "function", "reason"}
				for _, rec := range records {
					if rrr, ok := rec["error"]; ok {
						record := rrr.(map[string]any)
						out := make(MapRecord)
						for key, val := range record {
							if utils.InList(key, keys) {
								out[key] = val
							}
						}
						printRecord(out, "---")
					} else {
						fmt.Println(rec)
					}
				}
			} else {
				fmt.Printf("HTTP response: %+v, error %v\n", string(data), err)
			}
		}
	}
}

// helper function to add parent information
func provAddParent(args []string) {
	data, err := readInput(args)
	var rec dbs.ParentRecord
	err = json.Unmarshal(data, &rec)
	exit("", err)

	rurl := fmt.Sprintf("%s/parent", _srvConfig.Services.DataBookkeepingURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	printResponse(resp, err)
}

// helper function to add file information
func provAddFile(args []string) {
	data, err := readInput(args)
	var rec dbs.FileRecord
	err = json.Unmarshal(data, &rec)
	exit("", err)

	rurl := fmt.Sprintf("%s/file", _srvConfig.Services.DataBookkeepingURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	printResponse(resp, err)
}

// helper function to add dataset information
func provAddDataset(args []string) {
	data, err := readInput(args)
	var rec dbs.DatasetRecord
	err = json.Unmarshal(data, &rec)
	exit("", err)

	rurl := fmt.Sprintf("%s/dataset", _srvConfig.Services.DataBookkeepingURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	printResponse(resp, err)
}

// helper function to delete dataset information
func provDeleteRecord(args []string) {
}

// helper function to provide usage of dbs option
func provUsage() {
	fmt.Println("foxden prov <ls|add> [options]")
	fmt.Println("options: provenance attributes like dataset(s), file(s) or")
	fmt.Sprintf("         --file=<file name>, --did=<dataset id>\n")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list all provenance records:")
	fmt.Println("foxden prov ls <datasets|files>")
	fmt.Println("\n# list all dataset records for specific DID")
	fmt.Println("foxden prov ls datasets --did=<DID>")
	fmt.Println("\n# list all file records for specific DID")
	fmt.Println("foxden prov ls files --did=<DID>")
	//     fmt.Println("\n# remove provenance data record:")
	//     fmt.Println("foxden prov rm <dataset|site|file>")
	fmt.Println("\n# add provenance dataset data record:")
	fmt.Println("foxden prov add <dataset.json>")
	fmt.Println("\n# add provenance parent data record:")
	fmt.Println("foxden prov add-parent <parent.json>")
	fmt.Println("\n# add provenance file data record:")
	fmt.Println("foxden prov add-file <file.json>")
}
func provCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prov",
		Short: "foxden provenance commands",
		Long:  "foxden provenance commands to access FOXDEN Provenance service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			file, _ := cmd.Flags().GetString("file")
			did, _ := cmd.Flags().GetString("did")
			if len(args) == 0 {
				provUsage()
			} else if args[0] == "ls" {
				// obtain valid access token
				accessToken()
				provListRecord(args, did, file)
			} else if args[0] == "add" {
				writeToken()
				provAddDataset(args)
			} else if args[0] == "add-file" {
				writeToken()
				provAddFile(args)
			} else if args[0] == "add-parent" {
				writeToken()
				provAddParent(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().String("did", "", "did to use")
	cmd.PersistentFlags().String("file", "", "file to use")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		provUsage()
		return nil
	})
	return cmd
}
