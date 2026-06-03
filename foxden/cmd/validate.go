package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/CHESSComputing/golib/beamlines"
	"github.com/spf13/cobra"
)

// ─────────────────────────────────────────────────────────────────────────────
// Schema types
// ─────────────────────────────────────────────────────────────────────────────

// SchemaRecord represents one field definition from the JSON schema file.
type SchemaRecord struct {
	Key         string `json:"key"`
	Type        string `json:"type"` // "string","int","float","bool","list","list_str","list_int","list_float"
	Optional    bool   `json:"optional"`
	Multiple    bool   `json:"multiple"` // true → value may be a JSON array of the base type
	Section     string `json:"section"`  // UI section label — not used in validation output
	Description string `json:"description"`
	Units       string `json:"units"`
	Placeholder string `json:"placeholder"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Validation result types
// ─────────────────────────────────────────────────────────────────────────────

// ValidationError describes a single field-level problem.
type ValidationError struct {
	Field  string // schema key
	Kind   string // "missing", "type_mismatch", "unexpected"
	Detail string // human-readable explanation
}

func (e ValidationError) String() string {
	return fmt.Sprintf("  [%s] %s: %s", e.Kind, e.Field, e.Detail)
}

// ValidationResult collects errors and warnings for one record.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
}

func (r *ValidationResult) addError(field, kind, detail string) {
	r.Errors = append(r.Errors, ValidationError{field, kind, detail})
}

func (r *ValidationResult) addWarning(field, kind, detail string) {
	r.Warnings = append(r.Warnings, ValidationError{field, kind, detail})
}

func (r *ValidationResult) OK() bool { return len(r.Errors) == 0 }

// print writes a structured, human-friendly summary to stdout.
func (r *ValidationResult) print(label string) {
	divider := strings.Repeat("─", 60)
	fmt.Println(divider)
	if r.OK() && len(r.Warnings) == 0 {
		fmt.Printf("✓  %s — validation passed\n", label)
		fmt.Println(divider)
		return
	}
	if r.OK() {
		fmt.Printf("⚠  %s — passed with %d warning(s)\n", label, len(r.Warnings))
	} else {
		fmt.Printf("✗  %s — validation FAILED (%d error(s), %d warning(s))\n",
			label, len(r.Errors), len(r.Warnings))
	}
	fmt.Println(divider)
	if len(r.Errors) > 0 {
		fmt.Println("Errors:")
		for _, e := range r.Errors {
			fmt.Println(e.String())
		}
	}
	if len(r.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range r.Warnings {
			fmt.Println(w.String())
		}
	}
	fmt.Println(divider)
}

// ─────────────────────────────────────────────────────────────────────────────
// File loading helpers
// ─────────────────────────────────────────────────────────────────────────────

func loadSchema(path string) ([]SchemaRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read schema file %q: %w", path, err)
	}
	var schema []SchemaRecord
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("schema file %q is not valid JSON: %w", path, err)
	}
	if len(schema) == 0 {
		return nil, fmt.Errorf("schema file %q contains no field definitions", path)
	}
	return schema, nil
}

// loadRecord accepts either a JSON object {} or a single-element array [{}].
func loadRecord(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read record file %q: %w", path, err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err == nil {
		return obj, nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("record file %q must be a JSON object or a single-element JSON array", path)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("record file %q contains an empty JSON array", path)
	}
	if len(arr) > 1 {
		return nil, fmt.Errorf("record file %q contains %d records; validate one record at a time", path, len(arr))
	}
	return arr[0], nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Type helpers
// ─────────────────────────────────────────────────────────────────────────────

// isListType returns true for schema types whose value is inherently a JSON
// array (list, array, list_str, list_int, list_float).  Such fields are valid
// even when multiple:false because the array IS the value, not multiple values.
func isListType(t string) bool {
	switch strings.ToLower(t) {
	case "list", "array", "list_str", "list_int", "list_float":
		return true
	}
	return false
}

// checkType returns a descriptive error when goVal's runtime type does not
// match the schema type string.
//
// Supported type strings:
//
//	string                     — JSON string
//	int | integer              — JSON number with no fractional part
//	float | number             — any JSON number
//	bool | boolean             — JSON boolean
//	list | array               — JSON array (elements unchecked)
//	list_str                   — JSON array of strings
//	list_int                   — JSON array of integers
//	list_float                 — JSON array of numbers
//	any | ""                   — no constraint
func checkType(goVal any, schemaType string) error {
	switch strings.ToLower(schemaType) {

	case "string":
		if _, ok := goVal.(string); !ok {
			return fmt.Errorf("got %T, want string", goVal)
		}

	case "int", "integer":
		f, ok := goVal.(float64)
		if !ok {
			return fmt.Errorf("got %T, want integer", goVal)
		}
		if f != math.Trunc(f) {
			return fmt.Errorf("got float %.6g, want integer", f)
		}

	case "float", "number":
		if _, ok := goVal.(float64); !ok {
			return fmt.Errorf("got %T, want number", goVal)
		}

	case "bool", "boolean":
		if _, ok := goVal.(bool); !ok {
			return fmt.Errorf("got %T, want bool", goVal)
		}

	case "list", "array":
		// Array required; element types are unchecked.
		if _, ok := goVal.([]any); !ok {
			return fmt.Errorf("got %T, want JSON array", goVal)
		}

	case "list_str":
		arr, ok := goVal.([]any)
		if !ok {
			return fmt.Errorf("got %T, want array of strings", goVal)
		}
		for i, elem := range arr {
			if _, ok := elem.(string); !ok {
				return fmt.Errorf("element [%d] got %T, want string", i, elem)
			}
		}

	case "list_int":
		arr, ok := goVal.([]any)
		if !ok {
			return fmt.Errorf("got %T, want array of integers", goVal)
		}
		for i, elem := range arr {
			f, ok := elem.(float64)
			if !ok {
				return fmt.Errorf("element [%d] got %T, want integer", i, elem)
			}
			if f != math.Trunc(f) {
				return fmt.Errorf("element [%d] got float %.6g, want integer", i, f)
			}
		}

	case "list_float":
		arr, ok := goVal.([]any)
		if !ok {
			return fmt.Errorf("got %T, want array of numbers", goVal)
		}
		for i, elem := range arr {
			if _, ok := elem.(float64); !ok {
				return fmt.Errorf("element [%d] got %T, want number", i, elem)
			}
		}

	case "any", "":
		// No constraint.

	default:
		// Unknown schema type — skip rather than fail; schema may be extended.
	}
	return nil
}

// baseType returns the scalar element type for list types, so that
// multiple:true fields can validate individual array elements correctly.
//
//	list_str   → "string"
//	list_int   → "int"
//	list_float → "float"
//	list/array → "any"   (elements unchecked)
//	anything else → returned unchanged (already a scalar type)
func baseType(t string) string {
	switch strings.ToLower(t) {
	case "list_str":
		return "string"
	case "list_int":
		return "int"
	case "list_float":
		return "float"
	case "list", "array":
		return "any"
	default:
		return t
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Core validation engine
// ─────────────────────────────────────────────────────────────────────────────
func validateProvenance(recFile, schemaFile string) {
	// Not implemented yet
}

// ─────────────────────────────────────────────────────────────────────────────
// Command-level functions
// ─────────────────────────────────────────────────────────────────────────────

func validateMeta(recFile, schemaFile string) {
	schema := beamlines.Schema{FileName: schemaFile}
	err := schema.Load()
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	record, err := loadRecord(recFile)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	err = schema.Validate(record)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Usage + cobra wiring
// ─────────────────────────────────────────────────────────────────────────────

func validateUsage() {
	fmt.Println("foxden validate <meta|prov> <record.json> [options]")
	fmt.Println()
	fmt.Println("Validate a FOXDEN JSON record against a schema file.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  meta   Validate a metadata record")
	fmt.Println("  prov   Validate a provenance record")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --schema string   Path to the JSON schema file (required)")
	fmt.Println()
	fmt.Println("Supported schema types:")
	fmt.Println("  string, int, float, bool")
	fmt.Println("  list / array          — JSON array (elements unchecked)")
	fmt.Println("  list_str              — JSON array of strings")
	fmt.Println("  list_int              — JSON array of integers")
	fmt.Println("  list_float            — JSON array of numbers")
	fmt.Println()
	fmt.Println("Exit codes:")
	fmt.Println("  0   Validation passed (warnings are non-fatal)")
	fmt.Println("  1   Validation failed or an error occurred")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("  # Validate a metadata record against the 3A schema:")
	fmt.Println("  foxden validate meta record.json --schema=/path/ID3A.json")
	fmt.Println()
	fmt.Println("  # Validate a provenance record:")
	fmt.Println("  foxden validate prov prov.json --schema=/path/prov_schema.json")
	fmt.Println()
}

func validateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate FOXDEN metadata or provenance records against a schema",
		Long:  "Validate FOXDEN metadata or provenance records against a JSON schema\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			schemaFile, _ := cmd.Flags().GetString("schema")

			if len(args) == 0 {
				validateUsage()
				return
			}

			action := args[0]
			if action != "meta" && action != "prov" {
				fmt.Printf("WARNING: unsupported subcommand %q — expected meta or prov\n", action)
				validateUsage()
				return
			}

			if len(args) < 2 {
				fmt.Printf("ERROR: missing record file argument\n")
				fmt.Printf("Usage: foxden validate %s <record.json> --schema=<schema.json>\n", action)
				os.Exit(1)
			}

			if schemaFile == "" {
				fmt.Println("ERROR: --schema is required")
				fmt.Printf("Usage: foxden validate %s %s --schema=<schema.json>\n", action, args[1])
				os.Exit(1)
			}

			switch action {
			case "meta":
				validateMeta(args[1], schemaFile)
			case "prov":
				validateProvenance(args[1], schemaFile)
			}
		},
	}

	cmd.PersistentFlags().String("schema", "", "path to the JSON schema file (required)")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		validateUsage()
		return nil
	})
	return cmd
}
