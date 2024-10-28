package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

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
	var dbCol string
	flag.StringVar(&dbCol, "dbCol", "", "mongodb collection")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("### update globus links")
	updateMetaRecords(uri, dbName, dbCol)
	log.Println("### update DIDs")
	updateDIDs(uri, dbName, dbCol)
	log.Println("### update BTRs")
	updateBTRs(uri, dbName, dbCol)
}

// function which updates MongoDB records
func updateMetaRecords(uri, dbName, dbCol string) {
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
			result, err := c.UpdateOne(ctx, filter, update)
			//             opts := options.Update().SetUpsert(true)
			//             result, err := c.UpdateOne(ctx, filter, update, opts)
			if err != nil {
				log.Fatal(err)
			}
			// Check how many documents were modified
			if result.MatchedCount == 0 {
				log.Println("No document found with the given did", did)
			} else if result.ModifiedCount > 0 {
				fmt.Printf("Successfully updated the document did: %s\n", did)
			}
		}
	}
}

// function which updates DIDs in MongoDB
func updateDIDs(uri, dbName, dbCol string) {
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
			var newBeamline []string
			if val, ok := rec["beamline"]; ok {
				var blines []string
				for _, b := range val.([]string) {
					blines = append(blines, strings.ToLower(b))
				}
				newBeamline = blines
			}
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
				"beamline":    newBeamline,
				"btr":         newBtr,
				"cycle":       newCycle,
				"sample_name": newSample,
			}}
			/*
				result, err := c.UpdateOne(ctx, filter, update)
				// Check how many documents were modified
				if result.MatchedCount == 0 {
					log.Println("No document found with the given did", did)
				} else if result.ModifiedCount > 0 {
					fmt.Printf("Updated did: %s => %s\n", did, newDid)
				}
			*/
			log.Printf("will update: filter=%+v update=%+v", filter, update)
		}
	}
}

// function which updates BTRs in MongoDB
func updateBTRs(uri, dbName, dbCol string) {
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
		/*
			result, err := c.UpdateOne(ctx, filter, update)
			// Check how many documents were modified
			if result.MatchedCount == 0 {
				log.Println("No document found with the given did", did)
			} else if result.ModifiedCount > 0 {
				fmt.Printf("Updated did: %s => %s\n", did, newDid)
			}
		*/
		log.Printf("will update: filter=%+v update=%+v", filter, update)
		log.Printf("update record did=%s btr %s => %s", did, btr, newBtr)
	}
}
