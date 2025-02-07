package main

import (
	"flag"
	"log"
)

func main() {
	var uri string
	flag.StringVar(&uri, "uri", "", "mongodb uri")
	var dbName string
	flag.StringVar(&dbName, "dbName", "", "mongodb dbname")
	var dbCol string
	flag.StringVar(&dbCol, "dbCol", "", "mongodb collection")
	var execute bool
	flag.BoolVar(&execute, "execute", false, "execute actions on MongoDB records")
	var updateGlobus bool
	flag.BoolVar(&updateGlobus, "updateGlobus", false, "update Globus links on MongoDB records")
	var updateDID bool
	flag.BoolVar(&updateDID, "updateDID", false, "update DIDs on MongoDB records")
	var updateBTR bool
	flag.BoolVar(&updateBTR, "updateBTR", false, "updateBTRs on MongoDB records")
	var updateSID bool
	flag.BoolVar(&updateSID, "updateSID", false, "updateSIDs on MongoDB records")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if updateGlobus {
		log.Println("### update globus links")
		updateGlobusLinks(uri, dbName, dbCol, execute)
	} else if updateDID {
		log.Println("### update DIDs")
		updateDIDs(uri, dbName, dbCol, execute)
	} else if updateBTR {
		log.Println("### update BTRs")
		updateBTRs(uri, dbName, dbCol, execute)
	} else if updateSID {
		log.Println("### update SIDs")
		updateSIDs(uri, dbName, dbCol, execute)
	}
}
