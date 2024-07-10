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
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	migrate(fileIn, fileOut)
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
