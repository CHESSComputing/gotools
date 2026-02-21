package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	mongo "github.com/CHESSComputing/golib/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// function which updates DIDs in MongoDB
func updateDIDs(uri, dbName, dbCol string, execute bool) {
	var err error
	var spec = map[string]any{}
	spec["btr"] = "kalra-4168-e"

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
		if did == "" {
			//fmt.Println(rec)
			beamline := rec["beamline"].(string)
			btr := rec["btr"].(string)
			cycle := rec["cycle"].(string)
			sample_name := filepath.Base(rec["spec_file"].(string))
			newDid := fmt.Sprintf("/beamline=%s/btr=%s/cycle=%s/sample_name=%s", beamline, btr, cycle, sample_name)
			newDid = strings.ToLower(newDid)
			filter := map[string]any{"sid": rec["sid"]}
			update := map[string]any{"$set": map[string]any{
				"did": newDid,
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
		if did != strings.ToLower(did) ||
			did != strings.ReplaceAll(did, "\n", "") ||
			did != strings.ReplaceAll(did, "kinigstein-4149-a", "kinigste-4149-a") {
			newDid := strings.ToLower(did)
			newDid = strings.ReplaceAll(newDid, "\n", "")
			newDid = strings.ReplaceAll(newDid, "kinigstein-4149-a", "kinigste-4149-a")
			filter := map[string]any{"did": did}
			update := map[string]any{"$set": map[string]any{
				"did": newDid,
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
