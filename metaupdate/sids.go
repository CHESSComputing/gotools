package main

import (
	"context"
	"fmt"
	"log"

	mongo "github.com/CHESSComputing/golib/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Update Scan Ids. Values should represent a timestamp in ns, not sec.
func updateSIDs(uri, dbName, dbCol string, execute bool) {
	var err error
	var spec map[string]any

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
		// skip records without sids
		val, ok := rec["sid"]
		if !ok {
			continue
		}
		sid := val.(float64)
		var newSid string
		if sid == float64(int(sid)) {
			newSid = fmt.Sprintf("%.0f", sid)
		} else {
			newSid = fmt.Sprintf("%.10f", sid)
		}
		update := map[string]any{"$set": map[string]any{
			"sid": newSid,
		}}
		filter := map[string]any{"sid": sid}
		if execute {
			result, err := c.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("ERROR: updating sid %s, error %v\n", sid, err)
			} else {
				// Check how many documents were modified
				if result.MatchedCount == 0 {
					log.Println("No document found with the given sid", sid)
				} else if result.ModifiedCount > 0 {
					fmt.Printf("Updated sid: %s => %s\n", sid, newSid)
				}
			}
		} else {
			log.Printf("will update: filter=%+v update=%+v", filter, update)
		}
	}
}
