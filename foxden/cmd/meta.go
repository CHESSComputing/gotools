package cmd

// CHESComputing foxden tool: meta-data module
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
	"os"
	"strings"
	"time"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
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
	rurl := fmt.Sprintf("%s/meta/%s", srvConfig.Config.Services.MetaDataURL, site)
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

// helper function to get meta-data records
func getMeta(user, query string, skeys []string, sorder, idx, limit int) ([]map[string]any, int, error) {
	// check if we got request from trusted client
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		// get trusted token and assign it to http write request
		if token, err := trustedUser(); err == nil {
			_httpReadRequest.Token = token
			defer func() {
				_httpReadRequest.Token = ""
			}()
		}
	}
	var records []map[string]any
	if query == "" {
		query = "{}"
	}
	rec := services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query, Idx: idx, Limit: limit, SortKeys: skeys, SortOrder: sorder},
	}

	data, err := json.Marshal(rec)
	rurl := fmt.Sprintf("%s/search", srvConfig.Config.Services.MetaDataURL)
	if os.Getenv("FOXDEN_VERBOSE") != "" {
		fmt.Println("FOXDEN query:", rurl)
	}
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("fail /search, unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from meta-data service", err)
	}
	err = json.Unmarshal(data, &records)

	//     var response services.ServiceResponse
	//     err = json.Unmarshal(data, &response)
	if err != nil {
		log.Println("response data", string(data))
		exit("Unable to unmarshal the data", err)
	}

	// get total number of records
	rec = services.ServiceRequest{
		Client:       "foxden",
		ServiceQuery: services.ServiceQuery{Query: query},
	}

	data, err = json.Marshal(rec)
	rurl = fmt.Sprintf("%s/count", srvConfig.Config.Services.MetaDataURL)
	resp, err = _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("fail /count, unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from meta-data service", err)
	}
	var nrecords int
	err = json.Unmarshal(data, &nrecords)
	if err != nil {
		exit("unable to unmarshal the data", err)
	}
	return records, nrecords, nil
}

func didMetaData() (string, string, string) {
	attrs := strings.Join(utils.DIDKeys(""), ",")
	sep := "/"
	div := "="
	if srvConfig.Config != nil {
		if srvConfig.Config.DID.Attributes != "" {
			attrs = srvConfig.Config.DID.Attributes
		}
		if srvConfig.Config.DID.Separator != "" {
			sep = srvConfig.Config.DID.Separator
		}
		if srvConfig.Config.DID.Divider != "" {
			div = srvConfig.Config.DID.Divider
		}
	}
	return attrs, sep, div
}

// helper function to provide usage of meta option
func metaUsage() {
	attrs, sep, div := didMetaData()
	fmt.Println("foxden meta <ls|rm|view> [options]")
	fmt.Println("foxden meta <add|amend> <file.json> {options}")
	fmt.Println("options: --schema=<schema> --did-attrs=<attrs> --did-sep=<separator> --did-div=<divider> --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list meta data records:")
	fmt.Println("foxden meta ls")
	fmt.Println("\n# list meta data records for specific range:")
	fmt.Println("foxden meta ls --idx=10 --limit=20")
	fmt.Println("\n# list all meta data records using specific sorting key(s) and order:")
	fmt.Println("foxden meta ls --sort-keys=date --sort-order=1")
	fmt.Println("\n# list specific meta-data record:")
	fmt.Println("foxden meta view <DID>")
	fmt.Println("\n# remove meta-data record:")
	fmt.Println("foxden meta rm <DID>")
	fmt.Println("\n# add meta-data record with given schema, file and did attributes which create a did value:")
	fmt.Printf("foxden meta add <file.json> --schema=<schema> --did-attrs=%s --did-sep=%s --did-div=%s\n", attrs, sep, div)
	fmt.Println("\n# the same as above since it is default values, schema is part of the record")
	fmt.Println("foxden meta add <file.json>")
	fmt.Println("\n# the same as above but provide json output, schema is part of the record")
	fmt.Println("foxden meta add <file.json> --json")
	fmt.Println("\n# amend record in Metadata record, schema is part of the record")
	fmt.Println("foxden meta amend <file.json>")
	fmt.Println("\n# show example of meta-data record")
	fmt.Println("foxden meta info")
}

// helper function to add meta data record
func metaAddRecord(user, schemaName, fname string, attrs, sep, div string, jsonOutput bool, update bool) {
	// check if we got request from trusted client
	if os.Getenv("FOXDEN_TRUSTED_CLIENT") != "" {
		// get trusted token and assign it to http write request
		if token, err := trustedUser(); err == nil {
			_httpWriteRequest.Token = token
			defer func() {
				_httpWriteRequest.Token = ""
			}()
		}
	}

	// check if given fname is a file
	_, err := os.Stat(fname)
	exit(fmt.Sprintf("unable to check file stat, file %s", fname), err)
	file, err := os.Open(fname)
	exit(fmt.Sprintf("unable to open file %s", fname), err)
	defer file.Close()
	data, err := io.ReadAll(file)
	exit(fmt.Sprintf("unable to read file %s", fname), err)
	var record map[string]any
	err = json.Unmarshal(data, &record)
	exit(fmt.Sprintf("unable to unmarshal data, file %s", fname), err)

	// add proper did
	did, ok := record["did"]
	if !ok || did == "" {
		did = utils.CreateDID(record, attrs, sep, div)
		record["did"] = did
	}

	// we need to create /meta/file/upload call using URL form
	var mrec services.MetaRecord
	if schemaName == "" {
		// extract record from the record
		if val, ok := record["schema"]; ok {
			schemaName = val.(string)
		}
	}
	mrec.Schema = schemaName
	if schemaName == "" {
		exit("schema is not provided", errors.New("No schema"))
	}
	// add user info to metadata record
	if _, ok := record["user"]; !ok {
		record["user"] = user
	}

	mrec.Record = record
	data, err = json.MarshalIndent(mrec, "", "  ")
	exit("unable to marshal data", err)
	rurl := fmt.Sprintf("%s", srvConfig.Config.Services.MetaDataURL)
	var resp *http.Response
	if update {
		resp, err = _httpWriteRequest.Put(rurl, "application/json", bytes.NewBuffer(data))
	} else {
		resp, err = _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	}
	msg := fmt.Sprintf("fail %s unable to fetch data from meta-data service", rurl)
	exit(msg, err)
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	exit("unable to read data from meta-data service", err)

	if jsonOutput {
		fmt.Printf(string(data))
		return
	}
	var response services.ServiceResponse
	err = json.Unmarshal(data, &response)
	exit("Unable to unmarshal the data", err)
	if response.Status == "ok" {
		fmt.Printf("SUCCESS: record was successfully added with did\n")
		fmt.Println(did)
	} else {
		// check if we got middleware error
		if response.HttpCode == 0 {
			fmt.Printf("ERROR: %s", string(data))
		} else {
			fmt.Printf("ERROR: failed to add record to MetaData service\n%+v", response.String())
		}
	}
}

// helper function to delete meta-data record
func metaDeleteRecord(user, did string, jsonOutput bool) {
	if did == "" {
		metaUsage()
		os.Exit(1)
	}
	token, err := deleteAccessToken()
	exit("", err)
	rurl := fmt.Sprintf("%s/record?did=%s&user=%s", srvConfig.Config.Services.MetaDataURL, did, user)
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
	if jsonOutput {
		fmt.Println(string(body))
		return
	}
	if response.Status == "ok" {
		fmt.Printf("SUCCESS: record %s was successfully removed\n", did)
	} else {
		fmt.Printf("WARNING: record %s failed to be removed\n", did)
	}

}

// helper funtion to list meta-data records
func metaListRecord(user, spec string, skeys []string, sorder, idx, limit int, jsonOutput bool) {
	records, nrecords, err := getMeta(user, spec, skeys, sorder, idx, limit)
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
		fmt.Println("---")
		fmt.Printf("did        : %v\n", r["did"])
		fmt.Printf("schema     : %v\n", r["schema"])
		fmt.Printf("cycle      : %v\n", r["cycle"])
		fmt.Printf("beamline   : %v\n", r["beamline"])
		fmt.Printf("btr        : %v\n", r["btr"])
		fmt.Printf("sample_name: %v\n", r["sample_name"])
		tstamp := "Not Available"
		if val, ok := r["date"]; ok {
			secondsSinceEpoch := int64(val.(float64))
			tstamp = time.Unix(secondsSinceEpoch, 0).Format(time.RFC3339)
		}
		fmt.Printf("date       : %v\n", tstamp)
		//         fmt.Printf("%+v", r)
	}
	fmt.Println("---")
	fmt.Printf("Showing %d-%d out of %d records, for more records use --idx/--limit options\n", idx, idx+limit, nrecords)
}

// helper function to print meta data records in Json format
func metaJsonRecord(user, did string, skeys []string, sorder, idx, limit int, jsonOutput bool) {
	query := "did:" + did
	if verbose > 0 {
		log.Println("### query", query)
	}
	records, nrecords, err := getMeta(user, query, skeys, sorder, idx, limit)
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
		fmt.Println("---")
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			exit("unable to marshal data", err)
		}
		fmt.Println(string(data))
	}
	fmt.Println("---")
	if verbose > 0 {
		fmt.Printf("Showing %d-%d out of %d records, for more records use --idx/--limit options\n", idx, idx+limit, nrecords)
	}
}

func metaCommand() *cobra.Command {
	attrs, sep, div := didMetaData()
	var schema string
	cmd := &cobra.Command{
		Use:   "meta",
		Short: "foxden MetaData commands",
		Long:  "foxden MetaData commands to access FOXDEN MetaData service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			schema, _ := cmd.Flags().GetString("schema")
			attrs, _ := cmd.Flags().GetString("did-attrs")
			sep, _ := cmd.Flags().GetString("did-sep")
			div, _ := cmd.Flags().GetString("did-div")
			sortKeys, _ := cmd.Flags().GetString("sort-keys")
			sortOrder, _ := cmd.Flags().GetInt("sort-order")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			idx, _ := cmd.Flags().GetInt("idx")
			limit, _ := cmd.Flags().GetInt("limit")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			var skeys []string
			if sortKeys != "" {
				for _, k := range strings.Split(sortKeys, ",") {
					skeys = append(skeys, k)
				}
			}
			if sortOrder == 0 {
				sortOrder = -1
			}
			if len(args) == 0 {
				metaUsage()
			} else if args[0] == "ls" {
				user, _ := getUserToken()
				if len(args) == 2 {
					metaListRecord(user, args[1], skeys, sortOrder, idx, limit, jsonOutput)
				} else {
					metaListRecord(user, "", skeys, sortOrder, idx, limit, jsonOutput)
				}
			} else if args[0] == "view" {
				user, _ := getUserToken()
				if len(args) == 2 {
					metaJsonRecord(user, args[1], skeys, sortOrder, idx, limit, jsonOutput)
				} else {
					metaJsonRecord(user, "", skeys, sortOrder, idx, limit, jsonOutput)
				}
			} else if args[0] == "add" {
				token, _ := writeToken()
				var fname string
				if len(args) == 2 {
					fname = args[1]
				} else {
					metaUsage()
					exit("please provide <file.json>", errors.New("no input file"))
				}
				user := getUserFromToken(token)
				metaAddRecord(user, schema, fname, attrs, sep, div, jsonOutput, false)
			} else if args[0] == "amend" {
				token, _ := writeToken()
				var fname string
				if len(args) == 2 {
					fname = args[1]
				} else {
					metaUsage()
					exit("please provide <file.json>", errors.New("no input file"))
				}
				user := getUserFromToken(token)
				metaAddRecord(user, schema, fname, attrs, sep, div, jsonOutput, true)
			} else if args[0] == "info" {
				recordInfo("metadata.json")
			} else if args[0] == "rm" {
				deleteToken()
				user, _ := getUserToken()
				if user == "" {
					exit("unable to get user name from token value", errors.New("unknown user"))
				}
				if len(args) != 2 {
					metaUsage()
					exit("please provide did", errors.New("no did"))
				}
				did := args[1]
				metaDeleteRecord(user, did, jsonOutput)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().String("schema", schema, "schema name (ID1A3, ID3A, ID4B)")
	cmd.PersistentFlags().String("did-attrs", attrs, "did attributes")
	cmd.PersistentFlags().String("did-sep", sep, "did separator")
	cmd.PersistentFlags().String("did-div", div, "did key-value divider")
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().String("sort-keys", "date", "sort key(s), if multiple keys separate them by comma (default: date)")
	cmd.PersistentFlags().Int("sort-order", -1, "sort order: 1 ascending, -1 desecnding (default)")
	cmd.PersistentFlags().Int("idx", 0, "start index, default 0")
	cmd.PersistentFlags().Int("limit", 100, "limit number of records to given value, default 100")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		metaUsage()
		return nil
	})
	return cmd
}
