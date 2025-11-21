package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SchemaField matches your JSON input structure
type SchemaField struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Optional    bool   `json:"optional"`
	Multiple    bool   `json:"multiple"`
	Section     string `json:"section"`
	Description string `json:"description"`
	Units       string `json:"units"`
	Placeholder string `json:"placeholder"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: schemafmt <schema.json>")
		os.Exit(1)
	}

	filename := os.Args[1]

	// -------------------------
	// Extract schema name
	// -------------------------
	base := filepath.Base(filename)       // ID1A3.json
	name := strings.TrimSuffix(base, filepath.Ext(base)) // ID1A3

	// alias: strip non-alphanumerics, make lowercase
	re := regexp.MustCompile(`[^0-9a-zA-Z]+`)
	alias := strings.ToLower(re.ReplaceAllString(name, "")) // e.g. 1a3

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var fields []SchemaField
	if err := json.Unmarshal(data, &fields); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	// -------------------------
	// Print title
	// -------------------------
	fmt.Printf("Schema %s as known as %s\n\n", name, alias)

	// -------------------------
	// Print each field
	// -------------------------
	for _, f := range fields {
		fmt.Println(formatDescription(f, name, alias))
		fmt.Println() // blank line between entries
	}
}

// formatDescription converts a field into your text format
func formatDescription(f SchemaField, schema string, alias string) string {
	var b strings.Builder

	// key + description
	b.WriteString(fmt.Sprintf("%s is %s", f.Key, strings.TrimSpace(f.Description)))

	// units (if provided)
	if f.Units != "" {
		b.WriteString(fmt.Sprintf(" measured in %s", f.Units))
	}

	// data type
	b.WriteString(fmt.Sprintf(" with %s data-type", f.Type))

	// optional/mandatory + schema name + alias
	if f.Optional {
		b.WriteString(fmt.Sprintf(" which is optional for schema %s (%s)", schema, alias))
	} else {
		b.WriteString(fmt.Sprintf(" and mandatory for schema %s (%s)", schema, alias))
	}

	// multiplicity
	if f.Multiple {
		b.WriteString(" and can contain multiple values")
	} else {
		b.WriteString(" and accepts only a single value")
	}

	return b.String()
}

