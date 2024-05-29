package main

import (
	"context"
	"flag"
	"log"

	mongo "github.com/CHESSComputing/golib/mongo"
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
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	migrate(readUri, readDBName, readCollection, writeUri, writeDBName, writeCollection, verbose)
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
