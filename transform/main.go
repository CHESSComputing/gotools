package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	utils "github.com/CHESSComputing/golib/utils"
)

func main() {
	var fileIn string
	flag.StringVar(&fileIn, "fileIn", "", "input file name")
	var fileOut string
	flag.StringVar(&fileOut, "fileOut", "", "input file name")
	var schema bool
	flag.BoolVar(&schema, "schema", false, "transform schema meta-data record")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if schema {
		migrate(fileIn, fileOut)
	} else {
		migrateRecord(fileIn, fileOut)
	}
}

// MetaSchema describes FOXDEN meta-data schema record
type MetaSchema struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional"`
	Multiple    bool   `json:"multiple"`
	Section     string `json:"section"`
	Description string `json:"description"`
	Units       string `json:"units"`
	Placeholder string `json:"placeholder"`
	Value       any    `json:"value,omitempty"`
}

func migrate(fileIn, fileOut string) {
	if fileIn == "" {
		log.Fatal("empty input file name")
	}
	if fileOut == "" {
		log.Fatal("empty output file name")
	}

	file, err := os.Open(fileIn)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	var records []MetaSchema
	err = json.Unmarshal(data, &records)
	if err != nil {
		log.Fatal(err)
	}
	var out []MetaSchema
	for _, rec := range records {
		rec.Key = utils.CamelCaseToSnakeCase(rec.Key)
		out = append(out, rec)
	}
	fout, err := os.Create(fileOut)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	data, err = json.MarshalIndent(out, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fout.Write(data)
}

func migrateRecord(fileIn, fileOut string) {
	if fileIn == "" {
		log.Fatal("empty input file name")
	}
	if fileOut == "" {
		log.Fatal("empty output file name")
	}

	file, err := os.Open(fileIn)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	var record map[string]any
	err = json.Unmarshal(data, &record)
	if err != nil {
		log.Println("record", string(data))
		log.Fatal(err)
	}
	nrec := make(map[string]any)
	for key, val := range record {
		nkey := utils.CamelCaseToSnakeCase(key)
		nrec[nkey] = val
	}
	fout, err := os.Create(fileOut)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()
	data, err = json.MarshalIndent(nrec, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fout.Write(data)
}
