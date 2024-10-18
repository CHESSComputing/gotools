package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/CHESSComputing/golib/globus"
	mongo "github.com/CHESSComputing/golib/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	var uri string
	flag.StringVar(&uri, "uri", "", "mongodb uri")
	var dbName string
	flag.StringVar(&dbName, "dbName", "", "mongodb dbname")
	var collection string
	flag.StringVar(&collection, "collection", "", "mongodb collection")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	updateMetaRecords(uri, dbName, collection, verbose)
}

// function which updates MongoDB records
func updateMetaRecords(uri, dbName, collection string, verbose bool) {
	var err error
	var spec bson.M

	// read records from readUri MongoDB
	records := []map[string]any{}
	mongodb := mongo.Connection{URI: uri}
	ctx := context.TODO()
	mongoClient := mongodb.Connect()
	c := mongoClient.Database(dbName).Collection(collection)
	opts := options.Find()
	cur, err := c.Find(ctx, spec, opts)
	if err != nil {
		log.Fatal(err)
	}
	cur.All(ctx, &records)

	for _, rec := range records {
		// skip records with globus link
		_, ok := rec["globus_link"]
		if ok {
			continue
		}

		// skip records without did's
		val, ok := rec["did"]
		if !ok {
			continue
		}
		did := val.(string)

		// otherwise we'll create new globus link and update the record
		pat := "CHESS Raw"
		if val, ok := rec["data_location_raw"]; ok {
			path := val.(string)
			gurl, err := globus.ChessGlobusLink(pat, path)
			if err != nil {
				log.Fatal(err)
			}
			opts := options.Update().SetUpsert(true)
			filter := bson.M{"did": did}
			nrec := make(map[string]any)
			nrec["globus_link"] = gurl

			update := bson.M{"$set": nrec}
			result, err := c.UpdateOne(ctx, filter, update, opts)
			if err != nil {
				log.Fatal(err)
			}
			// Check how many documents were modified
			if result.MatchedCount == 0 {
				log.Println("No document found with the given ID", did)
			} else if result.ModifiedCount > 0 {
				fmt.Printf("Successfully updated the document with ID: %s\n", did)
			}
		}
	}
}
