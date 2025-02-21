package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	mongo "github.com/CHESSComputing/golib/mongo"
	sqldb "github.com/CHESSComputing/golib/sqldb"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var uri string
	flag.StringVar(&uri, "uri", "", "mongodb uri")
	var dbName string
	flag.StringVar(&dbName, "dbName", "", "mongodb dbname")
	var dbCol string
	flag.StringVar(&dbCol, "dbCol", "", "mongodb colection")
	var dbFileOld string
	flag.StringVar(&dbFileOld, "dbFileOld", "", "old motorsdb file")
	var dbFileNew string
	flag.StringVar(&dbFileNew, "dbFileNew", "", "new motorsdb file")
	var dbTypeOld string
	flag.StringVar(&dbTypeOld, "dbTypeOld", "", "old motorsdb type")
	var dbTypeNew string
	flag.StringVar(&dbTypeNew, "dbTypeNew", "", "new motorsdb type")
	var execute bool
	flag.BoolVar(&execute, "execute", false, "execute flag")

	flag.Parse()
	//addNewSids(uri, dbName, collection, dbFile, true)
	migrate(uri, dbName, dbCol, dbFileOld, dbFileNew, dbTypeOld, dbTypeNew, execute)
}

func migrate(uri, dbName, dbCol, dbFileOld, dbFileNew, dbTypeOld, dbTypeNew string, execute bool) {
	// oldSids := updateMongoSids(uri, dbName, dbCol, execute)
	oldSids := getOldSids(uri, dbName, dbCol)
	log.Printf("migrating %v records", len(oldSids))
	updateSql(dbFileOld, dbFileNew, dbTypeOld, dbTypeNew, oldSids, execute)
}

func convertSid(sid float64) string {
	if sid < 1e18 {
		sid = sid * 1e9
	}
	return strconv.Itoa(int(sid))
}

// get all old_sids from mongodb
func getOldSids(uri, dbName, dbCol string) []float64 {
	var oldSids []float64
	// read records from readUri MongoDB
	var spec map[string]any
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
	log.Printf("got %v records from mongodb", len(records))
	for _, rec := range records {
		val, ok := rec["sid_old"]
		if !ok {
			fmt.Printf("missing old_sid: %+v\n", rec)
			val, ok = rec["ScanId"]
			fmt.Printf("ScanId value instead: %+v\n", val)
			continue
		}
		sid := val.(float64)
		oldSids = append(oldSids, sid)
	}
	return oldSids
}

// update data type of sid in mongo db, return list of all old sids in the db.
func updateMongoSids(uri, dbName, dbCol string, execute bool) []float64 {
	var oldSids []float64
	// read records from readUri MongoDB
	var spec map[string]any
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
	log.Printf("got %v records from mongodb", len(records))
	for _, rec := range records {
		update := map[string]any{}
		filter := map[string]any{}
		val, ok := rec["sid"]
		var sid float64
		if !ok {
			// This record's keys aren't in snake_case,
			// and ScanId is int64.
			rename := map[string]string{}
			update["$rename"] = map[string]string{}
			for key := range rec {
				if key == "ScanId" || key == "_id" {
					// ScanId will be $set as "sid"
					// not $renamed.
					// Do not change _id in any way.
					continue
				}
				rename[key] = convertKey(key)
			}
			update["$rename"] = rename
			val = rec["ScanId"]
			int_sid := val.(int64)
			filter = map[string]any{"ScanId": int_sid}
			sid = float64(int_sid)
		} else {
			sid = val.(float64)
			filter = map[string]any{"sid": sid}
		}
		oldSids = append(oldSids, sid)
		newSid := convertSid(sid)
		update["$set"] = map[string]any{"sid": newSid, "sid_old": sid}
		if execute {
			// update mondogb
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
	return oldSids
}

// transfer all records from old to new db, performing conversion of sid data type along the way.
func updateSql(dbFileOld, dbFileNew, dbTypeOld, dbTypeNew string, oldSids []float64, execute bool) {
	// get old records
	oldSqlDb, err := sqldb.InitDB(dbTypeOld, dbFileOld)
	if err != nil {
		log.Println("could not init old sql db")
		return
	}
	tx, err := oldSqlDb.Begin()
	if err != nil {
		log.Fatal(err)
	}
	placeholders := strings.Repeat("?,", len(oldSids))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(`
    SELECT S.sid, group_concat(M.motor_mne), group_concat(P.motor_position)
    FROM MotorPositions AS P
    JOIN MotorMnes AS M ON M.motor_id=P.motor_id
    JOIN ScanIds AS S ON S.scan_id=M.scan_id
    WHERE S.scan_id IN (
        SELECT S.scan_id
        FROM MotorPositions AS P
        JOIN MotorMnes AS M ON M.motor_id=P.motor_id
        JOIN ScanIds AS S ON S.scan_id=M.scan_id
        WHERE S.sid IN (%s)
    )
    GROUP BY S.sid`, placeholders)
	args := make([]interface{}, len(oldSids))
	for i, sid := range oldSids {
		args[i] = sid
	}
	rows, err := tx.Query(query, args...)
	if err != nil {
		log.Fatal(err)
	}
	motorRecords := parseMotorRecords(rows)
	tx.Commit()
	// add records to new db
	newSqlDb, err := sqldb.InitDB(dbTypeNew, dbFileNew)
	if err != nil {
		log.Println("could not init new sql db")
		return
	}
	for _, rec := range motorRecords {
		if execute {
			_, err := InsertMotors(rec, newSqlDb)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("will add record with sid: %+v\n", rec.ScanId)
		}
	}
}

type MotorRecord struct {
	ScanId string
	Motors map[string]float64
}

func InsertMotors(r MotorRecord, db *sql.DB) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	// Insert the given motor record to the three tables that compose the static
	// motor positions database.
	log.Printf("Inserting motor record: %v", r)
	result, err := tx.Exec("INSERT INTO ScanIds (sid) VALUES (?)", r.ScanId)
	if err != nil {
		log.Printf("Could not insert record to ScanIds table; error: %v", err)
		return -1, err
	}
	scan_id, err := result.LastInsertId()
	if err != nil {
		log.Printf("Could not get ID of new record in ScanIds; error: %v", err)
		return scan_id, err
	}
	var motor_id int64
	for mne, pos := range r.Motors {
		result, err = tx.Exec("INSERT INTO MotorMnes (scan_id, motor_mne) VALUES (?, ?)", scan_id, mne)
		if err != nil {
			log.Printf("Could not insert record to MotorMnes table; error: %v", err)
			continue
		}
		motor_id, err = result.LastInsertId()
		if err != nil {
			log.Printf("Could not get ID of new record in MotorMnes; error: %v", err)
			continue
		}
		result, err = tx.Exec("INSERT INTO MotorPositions (motor_id, motor_position) VALUES (?, ?)", motor_id, pos)
		if err != nil {
			log.Printf("Could not insert record to MotorPositions table; error: %v", err)
		}
	}
	err = tx.Commit()
	return scan_id, err
}

func parseMotorRecords(rows *sql.Rows) []MotorRecord {
	// Helper for parsing grouped results of sql query
	var motor_records []MotorRecord
	// Parse the first record;
	// need to do this outside the loop if there is only one row of results.
	rows.Next()
	motor_record := parseMotorRecord(rows)
	motor_records = append(motor_records, motor_record)
	for rows.Next() {
		motor_record := parseMotorRecord(rows)
		motor_records = append(motor_records, motor_record)
	}
	return motor_records
}
func parseMotorRecord(rows *sql.Rows) MotorRecord {
	// Helper for parsing grouped results of sql query at the current cursor position only
	motor_record := MotorRecord{}
	var old_sid float64
	_motor_mnes, _motor_positions := "", ""
	err := rows.Scan(&old_sid, &_motor_mnes, &_motor_positions)
	if err != nil {
		log.Printf("Could not get a MotorRecord from a row of SQL results. error: %v", err)
		return motor_record
	}
	motor_mnes := strings.Split(_motor_mnes, ",")
	motor_positions := strings.Split(_motor_positions, ",")
	motors := make(map[string]float64)
	for i := 0; i < len(motor_mnes); i++ {
		motors[motor_mnes[i]], _ = strconv.ParseFloat(motor_positions[i], 64)
	}
	motor_record.ScanId = convertSid(old_sid)
	motor_record.Motors = motors
	return motor_record
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func convertKey(key string) string {
	if key == "ScanId" {
		return "sid"
	} else if key == "DatasetId" {
		return "did"
	} else if key == "Station" {
		return "beamline"
	}
	snake := matchFirstCap.ReplaceAllString(key, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
