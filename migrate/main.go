package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	authz "github.com/CHESSComputing/golib/authz"
	srvConfig "github.com/CHESSComputing/golib/config"
	mongo "github.com/CHESSComputing/golib/mongo"
	services "github.com/CHESSComputing/golib/services"
	utils "github.com/CHESSComputing/golib/utils"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	var readUri string
	flag.StringVar(&readUri, "readUri", "", "mongodb read uri")
	var readDBName string
	flag.StringVar(&readDBName, "readDBName", "", "mongodb read dbname")
	var readCollection string
	flag.StringVar(&readCollection, "readCollection", "", "mongodb read collection")
	var writeUri string
	flag.StringVar(&writeUri, "writeUri", "", "mongodb write uri")
	var writeDBName string
	flag.StringVar(&writeDBName, "writeDBName", "", "mongodb write dbname")
	var writeCollection string
	flag.StringVar(&writeCollection, "writeCollection", "", "mongodb write collection")
	var writeProvenance string
	flag.StringVar(&writeProvenance, "writeProvenance", "", "provenance uri")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if writeProvenance != "" {
		provenance(readUri, readDBName, readCollection, writeProvenance, verbose)
		return
	}
	migrate(readUri, readDBName, readCollection, writeUri, writeDBName, writeCollection, verbose)
}

// ProvRecord represents provenance input record
type ProvRecord struct {
	Buckets []string `json:"buckets"`
	Files   []string `json:"files"`
	Did     string   `json:"did"`
	Site    string   `json:"site"`
}

// helper function to add provenance information
func provenance(readUri, readDBName, readCollection, provUri string, verbose bool) {
	// get FOXDEN configuration
	hdir := os.Getenv("HOME")
	cfgFile := fmt.Sprintf("%s/.foxden.yaml", hdir)
	config, err := srvConfig.ParseConfig(cfgFile)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	srvConfig := &config

	// initialize FOXDEN writer
	httpWriteRequest := services.NewHttpRequest("write", 0)
	if httpWriteRequest.Token == "" {
		token := utils.ReadToken(os.Getenv("FOXDEN_WRITE_TOKEN"))
		if token == "" {
			fmt.Println("Please obtain write access token and put it into FOXDEN_WRITE_TOKEN env or file")
			os.Exit(1)
		}
		_, err := authz.TokenClaims(token, srvConfig.Authz.ClientID)
		if err != nil {
			fmt.Println("unable to use write token claims\nPlease check FOXDEN_WRITE_TOKEN env and set it up with token from 'foxden token create write' command", err)
			os.Exit(1)
		}
		httpWriteRequest.Token = token
	}

	// read all records from readUri
	records := []map[string]any{}
	readMongo := mongo.Connection{URI: readUri}
	readctx := context.TODO()
	readClient := readMongo.Connect()
	c := readClient.Database(readDBName).Collection(readCollection)
	opts := options.Find()
	var spec bson.M
	cur, err := c.Find(readctx, spec, opts)
	if err != nil {
		log.Fatal(err)
	}
	cur.All(readctx, &records)

	// loop over records and construct provenance ones
	for _, rec := range records {
		if val, ok := rec["data_location_raw"]; ok {
			rdir := fmt.Sprintf("%v", val)
			var files []string
			err := filepath.Walk(rdir,
				func(path string, _ os.FileInfo, err error) error {
					if strings.Contains(path, "/tmp") {
						if verbose {
							log.Println("skip", path)
						}
						return nil
					}
					if err != nil {
						return err
					}
					fileInfo, err := os.Stat(path)
					if err != nil {
						return err
					}
					if !fileInfo.IsDir() {
						files = append(files, path)
					}
					return nil
				})
			if err != nil {
				log.Println("ERROR: unable to read directory", rdir, "error: ", err)
				os.Exit(1)
			}
			if did, ok := rec["did"]; ok {
				// create provenance record to upload
				rec := ProvRecord{
					Did:   fmt.Sprintf("%v", did),
					Files: files,
					Site:  "Cornell",
				}
				data, err := json.Marshal(rec)
				if err != nil {
					log.Println("ERROR: unable to marshal record", rec, "error", err)
					os.Exit(1)
				}

				// update provenance information
				rurl := fmt.Sprintf("%s/dataset", srvConfig.Services.DataBookkeepingURL)
				resp, err := httpWriteRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
				if err != nil {
					log.Println("unable to send HTTP request to provenance service", err)
					os.Exit(1)
				}
				if resp.StatusCode == 200 {
					fmt.Printf("SUCCESS, did=%v provenance info is updated\n", did)
				} else {
					fmt.Printf("ERROR: did=%v provenance info is not updated, HTTP response %+v\n", did, resp)
				}
			}
		}
	}
}

// function which migrates records from one (reader) MongoDB URI/DB/Collection to
// another (output) MongoDB URI/DB/Collection
func migrate(readUri, readDBName, readCollection, writeUri, writeDBName, writeCollection string, verbose bool) {
	var err error
	var spec bson.M

	// read records from readUri MongoDB
	records := []map[string]any{}
	readMongo := mongo.Connection{URI: readUri}
	readctx := context.TODO()
	readClient := readMongo.Connect()
	c := readClient.Database(readDBName).Collection(readCollection)
	opts := options.Find()
	cur, err := c.Find(readctx, spec, opts)
	if err != nil {
		log.Fatal(err)
	}
	cur.All(readctx, &records)

	// transform and write records to writeUri MongoDB
	writeMongo := mongo.Connection{URI: writeUri}
	writectx := context.TODO()
	writeClient := writeMongo.Connect()
	c = writeClient.Database(writeDBName).Collection(writeCollection)
	for _, rec := range records {
		skip := false
		for _, key := range []string{"Beamline", "BTR", "Cycle", "SampleName"} {
			if _, ok := rec[key]; !ok {
				if verbose {
					log.Printf("skip record: %+v, no key %s", rec, key)
				}
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		// create proper did
		did := utils.GetDid(rec)
		if verbose {
			log.Printf("migrate record %v to %v", rec["did"], did)
		}
		// delete keys not used by FOXDEN
		for _, k := range []string{"_id", "path", "dataset"} {
			delete(rec, k)
		}
		rec["did"] = did
		opts := options.Update().SetUpsert(true)
		filter := bson.M{
			"beamline":    rec["Beamline"],
			"btr":         rec["BTR"],
			"cycle":       rec["Cycle"],
			"sample_name": rec["SampleName"],
		}

		// perform conversion from CamelCase to camel_case
		nrec := utils.ConvertCamelCaseKeys(rec)

		update := bson.M{"$set": nrec}
		if _, err := c.UpdateOne(writectx, filter, update, opts); err != nil {
			log.Fatal(err)
		}
	}
}
