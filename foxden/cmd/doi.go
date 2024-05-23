package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	utils "github.com/CHESSComputing/golib/utils"
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
	Files  []File       `json:"files,omitempty"`
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

// helper function to get did from given args
func getDID(sdid string) int64 {
	//     if len(args) < 2 {
	//         fmt.Println("ERROR: wrong number of arguments in getDID", args, len(args))
	//         os.Exit(1)
	//     }
	did, err := strconv.Atoi(sdid)
	exit("unable to parse did parameter", err)
	return int64(did)
}

// helper function to parget input args
func getParams(args []string) (int64, string) {
	if len(args) != 2 {
		fmt.Println("ERROR: wrong number of arguments in getParams", args, len(args))
		os.Exit(1)
	}
	did := getDID(args[0]) // first arugment should be id string
	fname := args[1]       // last argument should be file name
	return did, fname
}

// helper function to provide doi usage info
func doiUsage() {
	fmt.Println("foxden doi <ls|create|update|publish|view> <DID> [options]")
	fmt.Println("options: file name")
	fmt.Println("\nExamples:")
	fmt.Println("\n# create new document (new document with some ID, e.g. 123456789, will be created)")
	fmt.Println("foxden doi create")
	fmt.Println("\n# the out of above command will be like")
	fmt.Println("      Document is created: id=123456789 URL=https://zenodo.org/deposit/123456789")
	fmt.Println("\n# create new document from given record:")
	fmt.Println("foxden doi create </path/record.json>")
	fmt.Println("\n# add file to document id:")
	fmt.Println("foxden doi add <id> </path/regular/file>")
	fmt.Println("\n# update document id with publish data record:")
	fmt.Println("foxden doi update <id> /path/record.json")
	fmt.Println("\n# publish document id:")
	fmt.Println("foxden doi publish <id>")
	fmt.Println("\n# list existing documents:")
	fmt.Println("foxden doi ls <id>")
	fmt.Println("\n# get details of document id:")
	fmt.Println("foxden doi view <id>")
	fmt.Println()
	fmt.Println("Here is example of record.json")
	record := `
{
    "files": [
        {"name": "file1.txt", "file": "/path/file1.txt"},
        {"name": "file2.txt", "file": "/path/file2.txt"}
    ],
    "metadata": {
        "publication_type": "article",
        "upload_type": "publication",
        "description": "Test FOXDEN publication",
        "keywords": ["bla", "foo"],
        "creators": [{"name": "First Last", "affiliation": "Zenodo"}],
        "title": "Test experiment"
    }
}`
	fmt.Println(record)
}

func printDoiRecord(rec map[string]any) {
	maxLen := 20
	if val, ok := rec["id"]; ok {
		key := utils.PaddedKey("id", maxLen)
		vvv := val.(float64)
		v := int64(vvv)
		fmt.Printf("%s: %v\n", key, v)
	}
	if val, ok := rec["links"]; ok {
		vvv := val.(map[string]any)
		if v, ok := vvv["html"]; ok {
			key := utils.PaddedKey("URL", maxLen)
			fmt.Printf("%s: %v\n", key, v)
		}
	}
}

// helper function to list existing documents
func doiDocs(args []string) {
	rurl := fmt.Sprintf("%s/docs", _srvConfig.Services.PublicationURL)
	if len(args) == 2 {
		rurl += fmt.Sprintf("/%s", args[1])
	}
	resp, err := _httpReadRequest.Get(rurl)
	exit("http error", err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	exit("response read error", err)
	if len(args) == 2 {
		var rec map[string]any
		err = json.Unmarshal(data, &rec)
		exit("unmarshal error", err)
		printDoiRecord(rec)
		return
	}
	var records []map[string]any
	err = json.Unmarshal(data, &records)
	exit("unmarshal error", err)
	for _, rec := range records {
		fmt.Println("---")
		printDoiRecord(rec)
	}
}

// helper function to load zenodo record from given file name
func loadRecord(fname string) (PublishRecord, error) {
	var rec PublishRecord
	file, err := os.Open(fname)
	if err != nil {
		return rec, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return rec, err
	}
	err = json.Unmarshal(data, &rec)
	if err != nil {
		return rec, err
	}
	err = rec.Validate()
	if err != nil {
		return rec, err
	}
	return rec, nil
}

// helper function to create new document in Zenodo
func doiCreate(args []string) {
	if len(args) != 1 {
		fmt.Println("ERROR: wrong number of arguments in doiCreate", args, len(args))
		os.Exit(1)
	}
	// create new DOI resource
	rurl := fmt.Sprintf("%s/create", _srvConfig.Services.PublicationURL)
	resp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer([]byte{}))
	exit("unable to make HTTP request to publication service", err)
	// caputre response and extract document id (did)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	exit("unable to read response body", err)
	if verbose > 0 {
		fmt.Println("### create response", string(data))
	}
	var doc zenodo.CreateResponse
	err = json.Unmarshal(data, &doc)
	exit("unable to unmarshal record", err)
	fmt.Printf("Document is created: id=%d URL=%s\n", doc.Id, doc.Links.Html)
}

// helper function to get bucket id for given did
func getBucketId(did int64) string {
	rurl := fmt.Sprintf("%s/docs/%d", _srvConfig.Services.PublicationURL, did)
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	var rec DoiRecord
	err = json.Unmarshal(data, &rec)
	exit("unable to unmarshal record", err)
	arr := strings.Split(rec.Links.Bucket, "/")
	bid := arr[len(arr)-1]
	return bid
}

// helper function to add regular file to zenodo document id
func doiAdd(did int64, filePath string) {
	bid := getBucketId(did)
	arr := strings.Split(filePath, "/")
	fileName := arr[len(arr)-1]
	err := addFile(bid, fileName, filePath)
	exit("fail to add new file", err)
	fmt.Printf("Added %v to document %d\n", fileName, did)
}

// helper function to update zenodo document meta-data
func doiUpdate(did int64, fname string) {
	rec, err := loadRecord(fname)
	exit("unable to load record", err)

	// add meta-data record
	err = rec.MetaData.Validate()
	exit("fail to validate meta-data record", err)
	mrec := MetaRecord{Metadata: rec.MetaData}
	data, err := json.Marshal(mrec)
	exit("unable to marshal meta-data record", err)
	if verbose > 0 {
		fmt.Printf("### metadata: %s\n", string(data))
	}
	rurl := fmt.Sprintf("%s/update/%d", _srvConfig.Services.PublicationURL, did)
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
	fmt.Printf("Document %v is updated\n", did)
}

// helper function to publish zenodo document id
func doiPublish(did int64) {
	// publish the record
	rurl := fmt.Sprintf("%s/publish/%d", _srvConfig.Services.PublicationURL, did)
	publishResp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer([]byte{}))
	if err != nil || (publishResp.StatusCode < 200 || publishResp.StatusCode >= 400) {
		fmt.Printf("ERROR: unable to publish record, response %s, error %v\n", publishResp, err)
		os.Exit(1)
	}
	defer publishResp.Body.Close()
	if verbose > 0 {
		data, err := io.ReadAll(publishResp.Body)
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
	data, err := io.ReadAll(docsResp.Body)
	exit("unable to read server response", err)

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
func doiView(did int64) {
	rurl := fmt.Sprintf("%s/docs/%d", _srvConfig.Services.PublicationURL, did)
	resp, err := _httpReadRequest.Get(rurl)
	exit("unable to place HTTP request", err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	exit("unable to read response", err)
	var rec map[string]any
	err = json.Unmarshal(data, &rec)
	exit("unable to unmarshal the data", err)
	data, err = json.MarshalIndent(rec, "", " ")
	exit("unable to marshal the data", err)
	fmt.Println(string(data))
}

func doiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doi",
		Short: "foxden doi command",
		Long:  "foxden doi command to access FOXDEN Publication service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				doiUsage()
			} else if args[0] == "ls" {
				accessToken()
				doiDocs(args)
			} else if args[0] == "create" {
				accessToken()
				writeToken()
				doiCreate(args[1:])
			} else if args[0] == "add" {
				accessToken()
				writeToken()
				did, fname := getParams(args[1:])
				doiAdd(did, fname)
			} else if args[0] == "update" {
				accessToken()
				writeToken()
				did, fname := getParams(args[1:])
				doiUpdate(did, fname)
			} else if args[0] == "publish" {
				accessToken()
				writeToken()
				did := getDID(args[1])
				doiPublish(did)
			} else if args[0] == "view" {
				accessToken()
				did := getDID(args[1])
				doiView(did)
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
