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

// function which updates DIDs in MongoDB
func updateDIDs(uri, dbName, dbCol string, execute bool) {
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

		if did != strings.ToLower(did) {
			newDid := strings.ToLower(did)
			var newBtr, newCycle, newSample string
			if val, ok := rec["btr"]; ok {
				newBtr = strings.ToLower(fmt.Sprintf("%v", val))
			}
			if val, ok := rec["cycle"]; ok {
				newCycle = strings.ToLower(fmt.Sprintf("%v", val))
			}
			if val, ok := rec["sample_name"]; ok {
				newSample = strings.ToLower(fmt.Sprintf("%v", val))
			}
			filter := bson.M{"did": did}
			update := bson.M{"$set": bson.M{
				"did":         newDid,
				"btr":         newBtr,
				"cycle":       newCycle,
				"sample_name": newSample,
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
			}
		}
	}
}