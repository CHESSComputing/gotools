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
)

// PublishRecord describes publicatin record
type PublishRecord struct {
	MetaData zenodo.MetaDataRecord `json:"metadata"`
	//     Files    []zenodo.File         `json:"files"`
	Files any `json:"files,omitempty"`
}

// Validate provides validation of our publish record
func (r *PublishRecord) Validate() error {
	if err := r.MetaData.Validate(); err != nil {
		return err
	}
	return nil
}

// helper function to initialize http request with user's zenodo access token
func initZenodoAccess(tkn string) {
	var ztoken string

	// here we define priority of using user's token
	// if user provide explicitly tkn string we'll use it first
	// then if user specifies ZENODO_TOKEN env we'll use it next
	// finally, if user will have Zenodo:AccessToken in his/her foxden config we'll use it
	if tkn != "" {
		ztoken = utils.ReadToken(tkn)
	} else if os.Getenv("ZENODO_TOKEN") != "" {
		ztoken = os.Getenv("ZENODO_TOKEN")
	} else if _srvConfig.DOI.Zenodo.AccessToken != "" {
		ztoken = _srvConfig.DOI.Zenodo.AccessToken
	}

	if ztoken == "" {
		return
	}

	// if we provided with zenodo token we will use it
	if _httpReadRequest.Headers == nil {
		_httpReadRequest.Headers = make(map[string][]string)
	}
	_httpReadRequest.Headers["ZenodoAccessToken"] = []string{ztoken}
	if _httpWriteRequest.Headers == nil {
		_httpWriteRequest.Headers = make(map[string][]string)
	}
	_httpWriteRequest.Headers["ZenodoAccessToken"] = []string{ztoken}
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

// helper function to list existing documents
func zenodoDocs(did int64) {
	rurl := fmt.Sprintf("%s/docs", _srvConfig.Services.PublicationURL)
	if did != 0 {
		rurl += fmt.Sprintf("/%d", did)
	}
	resp, err := _httpReadRequest.Get(rurl)
	exit("http error", err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	exit("response read error", err)
	if did != 0 {
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
func zenodoCreate(fname string) {
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

	// update document if it was requested
	if doc.Id != 0 {
		if _, err := os.Stat(fname); err == nil {
			zenodoUpdate(doc.Id, fname)
		}
	}
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
	var rec zenodo.DoiRecord
	err = json.Unmarshal(data, &rec)
	exit("unable to unmarshal record", err)
	arr := strings.Split(rec.Links.Bucket, "/")
	bid := arr[len(arr)-1]
	return bid
}

// helper function to add regular file to zenodo document id
func zenodoAdd(did int64, filePath string) {
	bid := getBucketId(did)
	arr := strings.Split(filePath, "/")
	fileName := arr[len(arr)-1]
	err := addFile(bid, fileName, filePath)
	exit("fail to add new file", err)
	fmt.Printf("Added %v to document %d\n", fileName, did)
}

// helper function to update zenodo document meta-data
func zenodoUpdate(did int64, fname string) {
	rec, err := loadRecord(fname)
	exit("unable to load record", err)

	// add meta-data record
	err = rec.MetaData.Validate()
	exit("fail to validate meta-data record", err)
	mrec := zenodo.MetaRecord{Metadata: rec.MetaData}
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
func zenodoPublish(did int64) {
	// publish the record
	rurl := fmt.Sprintf("%s/publish/%d", _srvConfig.Services.PublicationURL, did)
	publishResp, err := _httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer([]byte{}))
	if err != nil || (publishResp.StatusCode < 200 || publishResp.StatusCode >= 400) {
		fmt.Printf("ERROR: unable to publish record, response %+v, error %v\n", publishResp, err)
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
	var doiRecord zenodo.DoiRecord
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
func zenodoView(did int64) {
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
