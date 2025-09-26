package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/CHESSComputing/golib/beamlines"
	srvConfig "github.com/CHESSComputing/golib/config"
)

// MetadataParameters holds all parameters we may need to generate metadata record
type MetadataParameters struct {
	Did    string
	Schema string
}

/*
// SchemaRecord defines one schema entry
type SchemaRecord struct {
	Key         string      `json:"key"`
	Type        string      `json:"type"`
	Optional    bool        `json:"optional"`
	Multiple    bool        `json:"multiple"`
	Section     string      `json:"section"`
	Description string      `json:"description"`
	Units       string      `json:"units"`
	Placeholder string      `json:"placeholder"`
	Value       any `json:"value,omitempty"`
}
*/

// GenerateRecord creates a JSON object from schema records
func GenerateRecord(schema []beamlines.SchemaRecord) (map[string]any, error) {
	record := make(map[string]any)

	for _, rec := range schema {
		// prefer Value over Placeholder
		var val any
		if rec.Value != nil {
			val = rec.Value
		} else {
			switch rec.Type {
			case "int":
				if rec.Placeholder != "" {
					if i, err := strconv.Atoi(rec.Placeholder); err == nil {
						val = i
					} else {
						val = 0
					}
				} else {
					val = 0
				}
			case "float64":
				if rec.Placeholder != "" {
					if f, err := strconv.ParseFloat(rec.Placeholder, 64); err == nil {
						val = f
					} else {
						val = 0.0
					}
				} else {
					val = 0.0
				}
			case "list_str":
				// ensure []string
				if rec.Value != nil {
					if arr, ok := rec.Value.([]any); ok {
						strs := make([]string, 0, len(arr))
						for _, v := range arr {
							if s, ok := v.(string); ok {
								strs = append(strs, s)
							}
						}
						val = strs
					} else {
						val = []string{}
					}
				} else if rec.Placeholder != "" {
					val = []string{rec.Placeholder}
				} else {
					val = []string{}
				}
			case "string":
				if rec.Placeholder != "" {
					val = rec.Placeholder
				} else {
					val = ""
				}
			default:
				val = rec.Placeholder
			}
		}

		record[rec.Key] = val
	}

	return record, nil
}

// helper function to generate metadata record
func generateMetadataRecord(rec MetadataParameters) {
	// fetch FOXDEN schemas
	rurl := fmt.Sprintf("%s/schemas", srvConfig.Config.Services.FrontendURL)
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch schemas from FOXDEN service", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from meta-data service", err)
	}
	var schemaRecords []map[string][]beamlines.SchemaRecord
	err = json.Unmarshal(data, &schemaRecords)

	// generate metadata record
	for _, schemaMap := range schemaRecords {
		if schema, ok := schemaMap[rec.Schema]; ok {
			record, _ := GenerateRecord(schema)
			out, _ := json.MarshalIndent(record, "", "  ")
			fmt.Println(string(out))
		}
	}
}
