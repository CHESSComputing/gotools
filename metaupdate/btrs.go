package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	mongo "github.com/CHESSComputing/golib/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

// function which updates BTRs in MongoDB
func updateBTRs(uri, dbName, dbCol string, execute bool) {
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
		// skip records without did's
		val, ok := rec["did"]
		if !ok {
			continue
		}
		did := val.(string)

		val, ok = rec["btr"]
		btr := val.(string)

		// if btr contains two dashes nothing needs to be done
		if len(strings.Split(btr, "-")) > 2 {
			continue
		}
		var newBtr string
		// extract from data_location_raw btr field
		if val, ok := rec["data_location_raw"]; ok {
			path := val.(string)
			arr := strings.Split(path, "/")
			for _, elem := range arr {
				if strings.Contains(elem, btr) {
					newBtr = elem
					break
				}
			}
		}
		// if we not found new btr we'll skip the record
		if newBtr == "" {
			continue
		}
		newDid := strings.Replace(did, btr, newBtr, -1)
		filter := bson.M{"did": did}
		update := bson.M{"$set": bson.M{
			"did": newDid,
			"btr": newBtr,
		}}
		if execute {
			result, err := c.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("ERROR: updating did %s, error %v\n", did, err)
			} else {
				// Check how many documents were modified
				if result.MatchedCount == 0 {
					log.Println("No document found with the given did", did)
				} else if result.ModifiedCount > 0 {
					fmt.Printf("Updated did: %s => %s\n", did, newDid)
				}
			}
		} else {
			log.Printf("will update: filter=%+v update=%+v", filter, update)
			log.Printf("update record did=%s btr %s => %s", did, btr, newBtr)
		}
	}
}
