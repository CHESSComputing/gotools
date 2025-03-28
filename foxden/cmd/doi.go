package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// DOIServiceResponse represents response structure of DOIService record
type DOIRecord struct {
	Doi            string `json:"doi"`
	DoiUrl         string `json:"doi_url"`
	Did            string `json:"did"`
	Description    string `json:"description"`
	Provider       string `json:"provider"`
	Published      int64  `json:"published"`
	Public         bool   `json:"public"`
	AccessMetadata bool   `json:"access_metadata"`
}

// helper function to fetch DOI records
func doiView(doi string, jsonOutput bool) {
	form := url.Values{}
	form.Set("doi", doi)

	// Convert form form to encoded format
	reqBody := bytes.NewBuffer([]byte(form.Encode()))
	// Create GET request to FOXDEN DOIService
	rurl := fmt.Sprintf("%s/search", srvConfig.Config.Services.DOIServiceURL)
	req, err := http.NewRequest("POST", rurl, reqBody)
	msg := fmt.Sprintf("fail %s unable to fetch data from FOXDEN DOIService", rurl)
	exit(msg, err)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	exit("unable to do HTTP request", err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	var records []DOIRecord
	err = json.Unmarshal(data, &records)
	exit("unable to read data from DOIService", err)
	if jsonOutput {
		fmt.Println(string(utils.FormatJsonRecords(data)))
		return
	}
	for _, rec := range records {
		fmt.Println("---")
		rtype := "Draft"
		if rec.Public {
			rtype = "Public"
		}
		fmt.Printf("doi : %s (%s)\n", rec.Doi, rtype)
		fmt.Printf("did : %s\n", rec.Did)
		fmt.Printf("url : %s\n", rec.DoiUrl)
		fmt.Printf("date: %s\n", time.Unix(rec.Published, 0).Format(time.RFC3339))
	}
}

// helper function to call FOXDEN publish form to publish DOI for a given set of parameters
func doiPublish(did, provider, description string, draft, metadata, jsonOutput bool) {
	// get FOXDEN metadata records for our did and extract schema from it
	user, _ := getUserToken()
	query := "did:" + did
	records, _, err := getMeta(user, query, []string{}, 0, 0, 1)
	exit(fmt.Sprintf("unable to meta-data record for did=%s", did), err)
	if len(records) != 1 {
		exit(fmt.Sprintf("multiple records found for did=%s", did), errors.New("multiple records"))
	}
	rec := records[0]
	var schema string
	if val, ok := rec["schema"]; ok {
		schema = val.(string)
	} else {
		exit(fmt.Sprintf("unable to identify schema for did=%s", did), errors.New("no schema"))
	}
	if description == "" {
		if val, ok := rec["description"]; ok {
			description = val.(string)
		}
	}

	// Define form data
	form := url.Values{}
	form.Set("did", did)
	form.Set("provider", provider)
	form.Set("description", description)
	form.Set("schema", schema)

	// Checkbox values should be sent as "on" if checked, otherwise omitted
	if draft {
		form.Set("draft", "on")
	}
	if metadata {
		form.Set("metadata", "on")
	}

	// Convert form form to encoded format
	reqBody := bytes.NewBuffer([]byte(form.Encode()))

	// Create POST request
	rurl := fmt.Sprintf("%s/publish", srvConfig.Config.Services.FrontendURL)
	hmap := make(map[string][]string)
	hmap["Accept"] = []string{"application/json"}
	_httpWriteRequest.Headers = hmap
	resp, err := _httpWriteRequest.Post(rurl, "application/x-www-form-urlencoded", reqBody)
	msg := fmt.Sprintf("fail %s unable to fetch data from FOXDEN Frontend service", rurl)
	exit(msg, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	exit("unable to read data from meta-data service", err)

	// Print response status
	fmt.Printf("Response Status: %v\n", resp.Status)
	if jsonOutput {
		fmt.Println(string(utils.FormatJson(data)))
	} else {
		var rec services.ServiceResponse
		if err := json.Unmarshal(data, &rec); err == nil {
			fmt.Println(rec.String())
		} else {
			fmt.Println(string(data))
		}
	}
}

// helper function to provide doi usage info
func doiUsage() {
	fmt.Println("foxden doi <ls|publish|view> [options]")
	fmt.Println("options:")
	fmt.Println("         <did> (dataset id)")
	fmt.Println("         --provider=<provider> (DOI provider: Datacite, Zenodo, MaterialCommons")
	fmt.Println("         --description=<description> (provide description about did)")
	fmt.Println("         --public (make DOI public record)")
	fmt.Println("         --hideMetadata (hide metadata from DOI publication)")
	fmt.Println("         --json (output in json data-format)")
	fmt.Println("\nExamples:")
	fmt.Println("\n# list documents from DOI provider:")
	fmt.Println("foxden doi ls <doi>")
	fmt.Println("\n# get details of document id:")
	fmt.Println("foxden doi view <doi>")
	fmt.Println("\n# publish metadata:")
	fmt.Println("foxden doi publish <did>")
	fmt.Println()
}

func doiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doi",
		Short: "foxden doi command",
		Long:  "foxden doi command to access FOXDEN Publication service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			//             tkn, _ := cmd.Flags().GetString("ztoken")
			provider, _ := cmd.Flags().GetString("provider")
			description, _ := cmd.Flags().GetString("description")
			publicDoi, _ := cmd.Flags().GetBool("public")
			hideMetadata, _ := cmd.Flags().GetBool("hideMetadata")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if len(args) == 0 {
				doiUsage()
			} else if args[0] == "ls" {
				var pat string
				if len(args) == 2 {
					pat = args[1]
				}
				doiView(pat, jsonOutput)
			} else if args[0] == "publish" {
				accessToken()
				writeToken()
				did := args[1]
				draft := !publicDoi
				publishMetadata := !hideMetadata
				doiPublish(did, provider, description, draft, publishMetadata, jsonOutput)
			} else if args[0] == "view" {
				accessToken()
				doi := args[1]
				doiView(doi, jsonOutput)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.PersistentFlags().String("provider", "Datacite", "DOI provider, default Datacite")
	cmd.PersistentFlags().String("description", "", "dataset description for DOI publication")
	cmd.PersistentFlags().Bool("public", false, "make public DOI")
	cmd.PersistentFlags().Bool("hideMetadata", false, "do not publish metadata in DOI publication")
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		doiUsage()
		return nil
	})
	return cmd
}
