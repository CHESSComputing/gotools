package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	mongo "github.com/CHESSComputing/golib/mongo"
	primitive "go.mongodb.org/mongo-driver/bson/primitive"
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
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	migrate(readUri, readDBName, readCollection, writeUri, writeDBName, writeCollection, verbose)
}

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
		did := getDid(rec)
		if verbose {
			log.Printf("migrate record %v to %v", rec["did"], did)
		}
		// delete keys not used by FOXDEN
		for _, k := range []string{"_id", "path", "dataset"} {
			delete(rec, k)
		}
		rec["did"] = did
		opts := options.Update().SetUpsert(true)
		filter := bson.M{"Beamline": rec["Beamline"], "BTR": rec["BTR"], "Cycle": rec["Cycle"], "SampleName": rec["SampleName"]}
		update := bson.M{"$set": rec}
		if _, err := c.UpdateOne(writectx, filter, update, opts); err != nil {
			log.Fatal(err)
		}
	}
}

func getValue(key string, rec map[string]any) string {
	var s string
	switch val := rec[key].(type) {
	case nil:
		s = ""
	case []string:
		var out []string
		for _, v := range val {
			out = append(out, fmt.Sprintf("%s", v))
		}
		s = strings.Join(out, ",")
	case primitive.A:
		var out []string
		for _, v := range val {
			out = append(out, fmt.Sprintf("%v", v))
		}
		s = strings.Join(out, ",")
	default:
		s = fmt.Sprintf("%v", val)
	}
	return s
}
func getDid(rec map[string]any) string {
	did := fmt.Sprintf("/beamline=%v/btr=%v/cycle=%v/sample=%v",
		getValue("Beamline", rec),
		getValue("BTR", rec),
		getValue("Cycle", rec),
		getValue("SampleName", rec),
	)
	return did
}
