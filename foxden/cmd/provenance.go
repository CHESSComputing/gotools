package cmd

// CHESComputing foxden tool: dbs module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	dbs "github.com/CHESSComputing/DataBookkeeping/dbs"
	srvConfig "github.com/CHESSComputing/golib/config"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// UrlParams represents all possible parameters we can pass to datasets query
type UrlParams struct {
	Did         string `url:"did"`
	File        string `url:"file"`
	Script      string `url:"script"`
	Environment string `url:"environment"`
	Package     string `url:"package"`
	Site        string `url:"site"`
	Bucket      string `url:"bucket"`
	Processing  string `url:"processing"`
	Osname      string `url:"osname"`
}

// helper function to construct Url
func buildUrl(rurl string, params UrlParams) string {
	baseURL, err := url.Parse(rurl)
	if err != nil {
		return rurl // Return original if parsing fails
	}

	query := url.Values{}
	val := reflect.ValueOf(params)
	typ := reflect.TypeOf(params)

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("url")
		value := fmt.Sprintf("%v", val.Field(i).Interface())

		if tag != "" && value != "" { // Ensure the field is not empty
			query.Set(tag, value)
		}
	}

	baseURL.RawQuery = query.Encode()
	return baseURL.String()
}

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

// helper function to list dataset information
func provListRecord(endpoint string, params UrlParams, jsonOutput bool) {
	rurl := fmt.Sprintf("%s/%s", srvConfig.Config.Services.DataBookkeepingURL, buildUrl(endpoint, params))
	for _, rec := range getData(rurl) {
		// convert seconds since epoch to human readable string
		if v, ok := rec["create_at"]; ok {
			if v != nil {
				rec["create_at"] = parseTimestamp(fmt.Sprintf("%v", v))
			}
		}
		if v, ok := rec["modify_at"]; ok {
			if v != nil {
				rec["modify_at"] = parseTimestamp(fmt.Sprintf("%v", v))
			}
		}
		if jsonOutput {
			data, err := json.Marshal(rec)
			if err == nil {
				fmt.Println(string(data))
			} else {
				exit("unable to marshal data record", err)
			}
		} else {
			// drop all _id fields to make more compact representation of the record
			nrec := make(MapRecord)
			for k, v := range rec {
				if strings.HasSuffix(k, "_id") {
					continue
				}
				nrec[k] = v
			}
			printRecord(nrec, "---")
		}
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
	data, err := readJsonData(lastArg)
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

	// first, we need to check if requested parent did exists in MetaData
	rurl := fmt.Sprintf("%s/record?did=%s", srvConfig.Config.Services.MetaDataURL, rec.Parent)
	resp, err := _httpReadRequest.Get(rurl)
	if resp.StatusCode != 200 {
		log.Println("### rurl ", rurl, "status code ", resp.StatusCode)
		err := errors.New("unable to find parent did")
		msg := fmt.Sprintf("For provided data=%+v there is no parent did=%s in MetaData service", rec, rec.Parent)
		exit(msg, err)
	}

	rurl = fmt.Sprintf("%s/parent", srvConfig.Config.Services.DataBookkeepingURL)
	resp, err = _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	printResponse(resp, err)
}

// helper function to add file information
func provAddFile(args []string) {
	data, err := readInput(args)
	var rec dbs.FileRecord
	err = json.Unmarshal(data, &rec)
	exit("", err)

	rurl := fmt.Sprintf("%s/file", srvConfig.Config.Services.DataBookkeepingURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	printResponse(resp, err)
}

// helper function to add dataset information
func provAddDataset(args []string, elapsedTime bool) {
	defer TrackTime("AddProvenance", elapsedTime)()
	data, err := readInput(args)
	var rec dbs.DatasetRecord
	err = json.Unmarshal(data, &rec)
	exit("unable to unmarshal provenance record", err)

	rurl := fmt.Sprintf("%s/dataset", srvConfig.Config.Services.DataBookkeepingURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))

	printResponse(resp, err)
}

// helper function to delete dataset information
func provDeleteRecord(args []string) {
}

// helper function to provide usage of dbs option
func provUsage() {
	fmt.Println("foxden prov <ls|add> [options]")
	fmt.Println("options: provenance attributes like dataset(s), file(s), parent(s), child(ren), etc.")
	fmt.Println("         --file=<file name>, --did=<dataset id>, --script=<script>")
	fmt.Println("         --site=<site name>, --bucket=<bucket name>")
	fmt.Println("         --environment=<environment name>, --package=<package name>")
	fmt.Println("         --processing=<processing name>, --osname=<os name>")
	fmt.Println("         --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# find provenance information for given DID using")
	fmt.Println("foxden prov ls provenance --did=<DID>")
	fmt.Println("\n# find provenance information for given DID using in JSON format, --json option can be applied to any command below")
	fmt.Println("foxden prov ls provenance --did=<DID> --json")
	fmt.Println("\n# find parts of provenance information for given DID using")
	fmt.Println("foxden prov ls datasets --did=<DID>")
	fmt.Println("foxden prov ls datasets --file=<filename>")
	fmt.Println("foxden prov ls datasets --osname=<os name>")
	fmt.Println("foxden prov ls datasets --environment=<environment>")
	fmt.Println("foxden prov ls datasets --script=<script>")
	fmt.Println("foxden prov ls datasets --file=<filename>")
	fmt.Println("foxden prov ls osinfo --did=<DID>")
	fmt.Println("foxden prov ls envronments --did=<DID>")
	fmt.Println("foxden prov ls scripts --did=<DID>")
	fmt.Println("foxden prov ls files --did=<DID>")
	fmt.Println("foxden prov ls parents --did=<DID>")
	fmt.Println("foxden prov ls child --did=<DID>")
	fmt.Println("\n# add provenance record:")
	fmt.Println("foxden prov add <provenance.json>")
	fmt.Println("\n# add provenance record using custom FOXDEN congregation (use foxden-dev instance)")
	fmt.Println("foxden prov add <provenance.json> --config=~/.foxden-dev.yaml")
	// fmt.Println("\n# add provenance parent data record:")
	// fmt.Println("foxden prov add-parent <parent.json>")
	// fmt.Println("\n# add provenance file data record:")
	// fmt.Println("foxden prov add-file <file.json>")
	// fmt.Println("\n# add provenance file data record but provide output in json format")
	// fmt.Println("foxden prov add-file <file.json> --json")
	fmt.Println("\n# show example of provenance record")
	fmt.Println("foxden prov info")
	fmt.Println("\n# generate provenance record")
	fmt.Println("foxden prov generate --inputDir /ipath --inputFilePattern \"*.jpg\" --outputDir /opath --did /a/b/c")
}

func provCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prov",
		Short: "foxden provenance commands",
		Long:  "foxden provenance commands to access FOXDEN Provenance service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			elapsedTime, _ := cmd.Flags().GetBool("elapsed-time")
			file, _ := cmd.Flags().GetString("file")
			did, _ := cmd.Flags().GetString("did")
			script, _ := cmd.Flags().GetString("script")
			environment, _ := cmd.Flags().GetString("environment")
			pkg, _ := cmd.Flags().GetString("pkg")
			site, _ := cmd.Flags().GetString("site")
			bucket, _ := cmd.Flags().GetString("bucket")
			processing, _ := cmd.Flags().GetString("processing")
			osname, _ := cmd.Flags().GetString("osname")
			inputDir, _ := cmd.Flags().GetString("inputDir")
			inputFilePattern, _ := cmd.Flags().GetString("inputFilePattern")
			outputDir, _ := cmd.Flags().GetString("outputDir")
			outputFilePattern, _ := cmd.Flags().GetString("outputFilePattern")
			params := UrlParams{
				Did:         did,
				File:        file,
				Script:      script,
				Environment: environment,
				Package:     pkg,
				Site:        site,
				Bucket:      bucket,
				Processing:  processing,
				Osname:      osname,
			}

			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			if len(args) == 0 {
				provUsage()
			} else if args[0] == "ls" {
				// obtain valid access token
				accessToken()
				if len(args) > 1 {
					endpoint := args[1]
					provListRecord(endpoint, params, jsonOutput)
				}
			} else if args[0] == "info" {
				recordInfo("provenance.json")
			} else if args[0] == "generate" {
				p := ProvenanceParameters{
					Did:      did,
					App:      "YOUR_APPLICATION",
					InputDir: inputDir, InputFilePattern: inputFilePattern,
					OutputDir: outputDir, OutputFilePattern: outputFilePattern,
				}
				generateProvenanceRecord(p)
			} else if args[0] == "add" {
				accessToken()
				writeToken()
				provAddDataset(args, elapsedTime)
				//             } else if args[0] == "add-file" {
				//                 accessToken()
				//                 writeToken()
				//                 provAddFile(args)
				//             } else if args[0] == "add-parent" {
				//                 accessToken()
				//                 writeToken()
				//                 provAddParent(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().String("did", "", "did to use")
	cmd.PersistentFlags().String("file", "", "file to use")
	cmd.PersistentFlags().String("script", "", "script to use")
	cmd.PersistentFlags().String("environment", "", "environment to use")
	cmd.PersistentFlags().String("package", "", "package to use")
	cmd.PersistentFlags().String("site", "", "site to use")
	cmd.PersistentFlags().String("bucket", "", "bucket to use")
	cmd.PersistentFlags().String("processing", "", "processing to use")
	cmd.PersistentFlags().String("osname", "", "osname to use")
	cmd.PersistentFlags().String("inputDir", "", "input directory to use")
	cmd.PersistentFlags().String("inputFilePattern", "", "file pattern to look in input directory")
	cmd.PersistentFlags().String("outputDir", "", "output directory to use")
	cmd.PersistentFlags().String("outputFilePattern", "", "file pattern to look in output directory")
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().Bool("elapsed-time", false, "print out elapsed time")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		provUsage()
		return nil
	})
	return cmd
}
