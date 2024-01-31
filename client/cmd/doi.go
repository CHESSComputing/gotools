package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/CHESSComputing/golib/zenodo"
	"github.com/spf13/cobra"
)

// File describes common file record
type File struct {
	Name string `json:"name"`
	File string `json:"file"`
}

// PublishRecord describes publicatin record
type PublishRecord struct {
	MetaData zenodo.MetaDataRecord `json:"metadata"`
	Files    []File                `json:"files"`
}

// MetaRecords used to publish meta-data record to zenodo
type MetaRecord struct {
	Metadata zenodo.MetaDataRecord `json:"metadata"`
}

// DoiRecord represents doi record
type DoiRecord struct {
	Id     int64        `json:"id"`
	Doi    string       `json:"doi"`
	DoiUrl string       `json:"doi_url"`
	Links  zenodo.Links `json:"links"`
}

// Validate provides validation of our publish record
func (r *PublishRecord) Validate() error {
	var msg string
	if len(r.Files) == 0 {
		msg = fmt.Sprintf("missing files")
	}
	if err := r.MetaData.Validate(); err != nil {
		return err
	}
	if msg != "" {
		return errors.New(msg)
	}
	return nil
}

// helper function to provide doi usage info
func doiUsage() {
	fmt.Println("client doi <ls|publish|view> [values]")
	fmt.Println("Examples:")
	fmt.Println("\n# publish new document:")
	fmt.Println("client doi publish /path/record.json")
	fmt.Println("\n# update document:")
	fmt.Println("client doi update /path/record.json")
	fmt.Println("\n# list existing documents:")
	fmt.Println("client doi ls")
	fmt.Println("\n# get details of document id:")
	fmt.Println("client doi view <id>")
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
		fmt.Println("---")
		printMap(rec)
	}
}

// helper function to publish new document
func doiPublish(args []string) {
	if len(args) != 2 {
		fmt.Println("ERROR: please provide JSON file with publish record")
		os.Exit(1)
	}
	verbose = 0 // TMP: for debugging

	// load our record
	fname := args[1]
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("ERROR: unable to read %s, error %v\n", fname, err)
		os.Exit(1)
	}
	var rec PublishRecord
	err = json.Unmarshal(data, &rec)
	if err != nil {
		fmt.Printf("ERROR: unable to unmarshal data record, error %v\n", err)
		os.Exit(1)
	}
	err = rec.Validate()
	if err != nil {
		fmt.Printf("ERROR: invalid publication record, error %v\n", err)
		os.Exit(1)
	}

	// create new DOI resource
	rurl := fmt.Sprintf("%s/create", _srvConfig.Services.PublicationURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer([]byte{}))
	if err != nil {
		fmt.Printf("ERROR: unable to make HTTP request to publication service, error %v\n", err)
		os.Exit(1)
	}
	// caputre response and extract document id (did)
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: unable to read response body, error %v", err)
		os.Exit(1)
	}
	if verbose > 0 {
		fmt.Println("### create response", string(data))
	}
	var doc zenodo.CreateResponse
	err = json.Unmarshal(data, &doc)
	if err != nil {
		fmt.Printf("ERROR: unable to unmarshal record, error %v", err)
		os.Exit(1)
	}
	did := doc.Id
	fmt.Println("New document has been created:", doc.Links.Html)
	// extract bucket id from bucket url
	// "https://zenodo.org/api/files/50b47f75-c97d-47c6-af11-caa6e967c1d5",
	arr := strings.Split(doc.Links.Bucket, "/")
	bid := arr[len(arr)-1]

	if verbose > 0 {
		fmt.Println("### did", did, "bid", bid)
	}

	// add files
	for _, r := range rec.Files {
		if err := addFile(bid, r.Name, r.File); err != nil {
			fmt.Println("ERROR", err)
			os.Exit(1)
		}
	}
	//     fmt.Printf("Added %d files to doi document\n", len(rec.Files))

	// add meta-data record
	err = rec.MetaData.Validate()
	if err != nil {
		fmt.Printf("ERROR: unable to marshal meta-data record, error %v\n", err)
		os.Exit(1)
	}
	mrec := MetaRecord{Metadata: rec.MetaData}
	data, err = json.Marshal(mrec)
	if err != nil {
		fmt.Printf("ERROR: unable to marshal meta-data record, error %v\n", err)
		os.Exit(1)
	}
	if verbose > 0 {
		fmt.Printf("### metadata: %s\n", string(data))
	}
	rurl = fmt.Sprintf("%s/update/%d", _srvConfig.Services.PublicationURL, did)
	metaResp, err := _httpWriteRequest.Put(rurl, "application/json", bytes.NewBuffer(data))
	defer metaResp.Body.Close()
	if verbose > 0 {
		data, err = io.ReadAll(metaResp.Body)
		fmt.Printf("### update %s response %s, error %v", rurl, string(data), err)
	}
	if err != nil || metaResp.StatusCode != 200 {
		fmt.Printf("ERROR: unable to add meta-data record, rsponse %s, error %v\n", metaResp, err)
		os.Exit(1)
	}
	//     fmt.Println("Added meta-data inforation to doi document")

	// publish the record
	rurl = fmt.Sprintf("%s/publish/%d", _srvConfig.Services.PublicationURL, did)
	publishResp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer([]byte{}))
	if err != nil || (publishResp.StatusCode < 200 || publishResp.StatusCode >= 400) {
		fmt.Printf("ERROR: unable to publish record, response %s, error %v\n", publishResp, err)
		os.Exit(1)
	}
	defer publishResp.Body.Close()
	if verbose > 0 {
		data, err = io.ReadAll(publishResp.Body)
		fmt.Printf("### publish %s response %s, error %v", rurl, string(data), err)
	}

	// fetch our document
	rurl = fmt.Sprintf("%s/docs/%d", _srvConfig.Services.PublicationURL, did)
	docsResp, err := _httpReadRequest.Get(rurl)
	if err != nil || (docsResp.StatusCode < 200 || docsResp.StatusCode >= 400) {
		fmt.Printf("ERROR: unable to fetch document, response %s, error %v\n", docsResp, err)
		os.Exit(1)
	}
	defer docsResp.Body.Close()
	data, err = io.ReadAll(docsResp.Body)
	if err != nil {
		fmt.Printf("ERROR: unable to read, response %s, error %v\n", docsResp, err)
		os.Exit(1)
	}
	//     fmt.Println("DOI document:")
	//     fmt.Println(string(data))

	// parse doi record
	var doiRecord DoiRecord
	err = json.Unmarshal(data, &doiRecord)
	if err == nil {
		fmt.Println("DOI", doiRecord.DoiUrl)
	}
	fmt.Println("Zenodo:", doiRecord.Links.Html)
}

// helper function to add file to our record
func addFile(bid, name, fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// create new DOI resource
	rurl := fmt.Sprintf("%s/add/%s/%s", _srvConfig.Services.PublicationURL, bid, name)
	resp, err := _httpWriteRequest.Put(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if verbose > 0 {
		data, err = io.ReadAll(resp.Body)
		fmt.Printf("### add file %s response %s, error %v\n", rurl, string(data), err)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 200 {
		msg := fmt.Sprintf("unable to add file, status %s", resp.Status)
		return errors.New(msg)
	}
	return nil
}

// helper function to view given doi document
func doiView(args []string) {
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
			} else if args[0] == "publish" {
				accessToken()
				writeToken()
				doiPublish(args)
			} else if args[0] == "view" {
				accessToken()
				doiView(args)
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
