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
	"net/http"
	"os"
	"strings"
	"time"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// helper function to get meta-data records
func tmplGet(murl, user string, idx, limit int) ([]map[string]any, int, error) {
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

	rurl := fmt.Sprintf("%s/tmpl/records", murl)
	if os.Getenv("FOXDEN_VERBOSE") != "" {
		fmt.Println("FOXDEN query:", rurl)
	}
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("fail /tmpl/records, unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from meta-data service", err)
	}
	err = json.Unmarshal(data, &records)
	return records, len(records), err
}

// helper function to provide usage of meta option
func tmplMetaUsage() {
	fmt.Println("foxden tmpl <ls|rm|view> [options]")
	fmt.Println("foxden tmpl <add|amend> <file.json> {options}")
	fmt.Println("options: --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list template metadata records:")
	fmt.Println("foxden tmpl ls")
	fmt.Println("\n# list meta data records for specific range:")
	fmt.Println("foxden tmpl ls --idx=10 --limit=20")
	fmt.Println("\n# list specific meta-data record:")
	fmt.Println("foxden tmpl view <DID>")
	fmt.Println("\n# remove meta-data record:")
	fmt.Println("foxden tmpl rm <DID>")
	fmt.Println("\n# the same as above since it is default values, tmpl_schema is part of the record")
	fmt.Println("foxden tmpl add <file.json>")
}

// helper function to add meta data record
func tmplMetaAddRecord(user string, data []byte, jsonOutput bool, update bool) {
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

	var record map[string]any
	err := json.Unmarshal(data, &record)
	exit("unable to unmarshal data", err)

	// add proper did
	did, ok := record["did"]
	if !ok || did == "" {
		attrs := strings.Join(utils.DIDKeys(""), ",")
		sep := "/"
		div := "="
		did = utils.CreateDID(record, attrs, sep, div)
		record["did"] = did
	}

	// we need to create /meta/file/upload call using URL form
	var schemaName string
	// extract record from the record
	if val, ok := record["tmpl_schema"]; ok {
		schemaName = val.(string)
	}
	if schemaName == "" {
		exit("unable to add tmpl record, tmpl_schema attribute is missing", errors.New("wrong template"))
	}
	// add user info to metadata record
	if _, ok := record["user"]; !ok {
		record["user"] = user
	}

	data, err = json.MarshalIndent(record, "", "  ")
	exit("unable to marshal data", err)
	rurl := fmt.Sprintf("%s/tmpl/record", srvConfig.Config.Services.MetaDataURL)
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
	exit("unable to unmarshal the metadata service response data", err)
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
func tmplMetaDeleteRecord(user, did string, jsonOutput bool) {
	if did == "" {
		tmplMetaUsage()
		os.Exit(1)
	}
	token, err := deleteAccessToken()
	exit("", err)
	rurl := fmt.Sprintf("%s/tmpl/record?did=%s&user=%s", srvConfig.Config.Services.MetaDataURL, did, user)
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
func tmplMetaListRecord(user, spec string, idx, limit int, jsonOutput bool) {
	rurl := srvConfig.Config.MetaDataURL
	records, nrecords, err := tmplGet(rurl, user, idx, limit)
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
		fmt.Printf("schema     : %v\n", r["tmpl_schema"])
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
func tmplMetaRecord(user string, idx, limit int, jsonOutput bool) {
	rurl := srvConfig.Config.MetaDataURL
	records, nrecords, err := tmplGet(rurl, user, idx, limit)
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

func tmplCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tmpl",
		Short: "foxden template metadata commands",
		Long:  "foxden template metadata commands to access FOXDEN MetaData service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			idx, _ := cmd.Flags().GetInt("idx")
			limit, _ := cmd.Flags().GetInt("limit")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			if len(args) == 0 {
				tmplMetaUsage()
			} else if args[0] == "ls" {
				user, _ := getUserToken()
				if len(args) == 2 {
					tmplMetaListRecord(user, args[1], idx, limit, jsonOutput)
				} else {
					tmplMetaListRecord(user, "", idx, limit, jsonOutput)
				}
			} else if args[0] == "view" {
				user, _ := getUserToken()
				tmplMetaRecord(user, idx, limit, jsonOutput)
			} else if args[0] == "add" {
				token, _ := writeToken()
				var fname string
				if len(args) == 2 {
					fname = args[1]
				} else {
					tmplMetaUsage()
					exit("please provide <file.json>", errors.New("no input file"))
				}
				user := getUserFromToken(token)
				data, err := readJsonData(fname)
				exit("unable to read data from input file", err)
				tmplMetaAddRecord(user, data, jsonOutput, false)
			} else if args[0] == "info" {
				recordInfo("tmpl_metadata.json")
			} else if args[0] == "rm" {
				deleteToken()
				user, _ := getUserToken()
				if user == "" {
					exit("unable to get user name from token value", errors.New("unknown user"))
				}
				if len(args) != 2 {
					tmplMetaUsage()
					exit("please provide did", errors.New("no did"))
				}
				did := args[1]
				tmplMetaDeleteRecord(user, did, jsonOutput)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Int("idx", 0, "start index, default 0")
	cmd.PersistentFlags().Int("limit", 100, "limit number of records to given value, default 100")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		tmplMetaUsage()
		return nil
	})
	return cmd
}
