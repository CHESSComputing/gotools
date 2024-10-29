package main

import (
	"context"
	"fmt"
	"log"

	"github.com/CHESSComputing/golib/globus"
	mongo "github.com/CHESSComputing/golib/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

// function which updates globus links MongoDB records
func updateGlobusLinks(uri, dbName, dbCol string, execute bool) {
	var err error
	var spec bson.M

	// read records from readUri MongoDB
	records := []map[string]any{}
	mongodb := mongo.Connection{URI: uri}
	ctx := context.TODO()
	mongoClient := mongodb.Connect()
	c := mongoClient.Database(dbName).Collection(dbCol)
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
				log.Printf("WARNING: skip %s, error: %v\n", did, err)
				continue
			}
			filter := bson.M{"did": did}
			update := bson.M{"$set": bson.M{"globus_link": gurl}}
			if execute {
				result, err := c.UpdateOne(ctx, filter, update)
				if err != nil {
					log.Printf("ERROR: updating did %s, error %v\n", did, err)
				} else {
					// Check how many documents were modified
					if result.MatchedCount == 0 {
						log.Println("No document found with the given did", did)
					} else if result.ModifiedCount > 0 {
						fmt.Printf("Successfully updated the document did: %s\n", did)
					}
				}
			} else {
				log.Println("update meta-data records with filter %+v and spec %+v", did, filter, update)
			}
		}
	}
}
