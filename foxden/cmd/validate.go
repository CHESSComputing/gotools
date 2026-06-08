package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CHESSComputing/golib/beamlines"
	"github.com/spf13/cobra"
)

// ─────────────────────────────────────────────────────────────────────────────
// File loading helpers
// ─────────────────────────────────────────────────────────────────────────────

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
	if report := schema.ValidateAll(record); report != "" {
		fmt.Println(report)
		os.Exit(1)
	}
	fmt.Println("✓ validation passed")
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
