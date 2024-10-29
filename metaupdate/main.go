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
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("### update globus links")
	updateGlobusLinks(uri, dbName, dbCol, execute)
	log.Println("### update DIDs")
	updateDIDs(uri, dbName, dbCol, execute)
	log.Println("### update BTRs")
	updateBTRs(uri, dbName, dbCol, execute)
}
