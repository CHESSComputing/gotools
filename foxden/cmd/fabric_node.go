package cmd

// CHESComputing foxden tool: fabric node module
//
// import (
import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// IngestRecord represents ingest record from FabricNode
type IngestRecord struct {
	Ingested int    `json:"ingested"`
	Did      string `json:"did"`
	GraphIRI string `json:"graphIRI"`
}

// helper function to provide fabric usage info
func fabricUsage() {
	fmt.Println("foxden fabric <ls|ingest|health|sparql|catalog> [options]")
	fmt.Println("\nExamples:")
	fmt.Println()
	fmt.Println("# list datasets for a beamline in the catalog:")
	fmt.Println("foxden fabric ls <beamline>")
	fmt.Println()
	fmt.Println("# ingest a single DID into FabricNode:")
	fmt.Println("foxden fabric ingest <did>")
	fmt.Println()
	fmt.Println("# ingest all DIDs from a file (one DID per line):")
	fmt.Println("foxden fabric ingest <dids-file>")
	fmt.Println()
	fmt.Println("# check health of data-service and catalog-service:")
	fmt.Println("foxden fabric health")
	fmt.Println()
	fmt.Println("# run SPARQL verification for a dataset (beamline is extracted from the DID):")
	fmt.Println("foxden fabric sparql <did>")
	fmt.Println()
	fmt.Println("# limit the number of triples shown in SPARQL output:")
	fmt.Println("foxden fabric sparql <did> --limit 10")
	fmt.Println()
	fmt.Println("# verify catalog entry for a beamline:")
	fmt.Println("foxden fabric catalog <beamline>")
}

// helper function to check health of FabricNode services
func fabricHealth(jsonOutput bool) {
	type healthResult struct {
		Service string `json:"service"`
		Status  string `json:"status"`
		Error   string `json:"error,omitempty"`
	}

	services := []struct {
		name string
		url  func() string
	}{
		{
			"catalog-service",
			func() string {
				return fmt.Sprintf("%s/health", srvConfig.Config.Services.FabricCatalogURL)
			},
		},
		{
			"data-service",
			func() string {
				return fmt.Sprintf("%s/health", srvConfig.Config.Services.FabricDataServiceURL)
			},
		},
	}

	var results []healthResult
	allOK := true

	for _, svc := range services {
		rurl := svc.url()
		if verbose > 0 {
			fmt.Println("HTTP GET", rurl)
		}
		resp, err := _httpReadRequest.Get(rurl)
		if err != nil {
			results = append(results, healthResult{Service: svc.name, Status: "error", Error: err.Error()})
			allOK = false
			continue
		}
		defer resp.Body.Close()

		var body map[string]any
		if decErr := json.NewDecoder(resp.Body).Decode(&body); decErr != nil {
			results = append(results, healthResult{Service: svc.name, Status: "error", Error: decErr.Error()})
			allOK = false
			continue
		}

		status := "error"
		if resp.StatusCode == 200 {
			if s, ok := body["status"].(string); ok && s == "ok" {
				status = "ok"
			}
		}
		r := healthResult{Service: svc.name, Status: status}
		if status != "ok" {
			allOK = false
			if msg, ok := body["error"].(string); ok {
				r.Error = msg
			} else {
				r.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
		}
		results = append(results, r)
	}

	if jsonOutput {
		if data, err := json.MarshalIndent(results, "", "  "); err == nil {
			fmt.Println(string(data))
		}
		return
	}

	for _, r := range results {
		if r.Status == "ok" {
			fmt.Printf("  ✓ %s healthy\n", r.Service)
		} else {
			fmt.Printf("  ✗ %s unhealthy: %s\n", r.Service, r.Error)
		}
	}
	if allOK {
		fmt.Println("\nAll services healthy.")
	} else {
		fmt.Println("\nOne or more services are unhealthy.")
		os.Exit(1)
	}
}

// helper function to run SPARQL verification for a dataset.
// The beamline is extracted from the DID itself.
func fabricSPARQL(args []string, jsonOutput bool, limit int) {
	if len(args) < 2 {
		fmt.Println("ERROR: sparql requires <did>")
		fmt.Println("Usage: foxden fabric sparql <did>")
		os.Exit(1)
	}
	did := args[1]

	bl := utils.GetBeamline(did)
	if bl == "" {
		fmt.Printf("ERROR: cannot extract beamline from DID %q\n", did)
		os.Exit(1)
	}

	encodedDid := url.PathEscape(did)
	rurl := fmt.Sprintf("%s/beamlines/%s/datasets/%s/sparql",
		srvConfig.Config.Services.FabricDataServiceURL, bl, encodedDid)
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}

	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("ERROR decoding response:", err)
		os.Exit(1)
	}

	if jsonOutput {
		if val, err := json.MarshalIndent(data, "", "  "); err == nil {
			fmt.Println(string(val))
		}
		return
	}

	// Print a human-readable SPARQL summary.
	fmt.Printf("SPARQL results for beamline=%s did=%s\n", bl, did)

	results, _ := data["results"].(map[string]any)
	if results == nil {
		fmt.Println("  No results block in response.")
		return
	}
	bindings, _ := results["bindings"].([]any)
	fmt.Printf("  %d triple(s) returned\n", len(bindings))

	if len(bindings) == 0 {
		fmt.Println("  WARNING: named graph is empty — ingest may have failed or DID not found.")
		return
	}

	// Show up to `limit` triples for quick inspection.
	show := limit
	if show <= 0 || show > len(bindings) {
		show = len(bindings)
	}
	for i, b := range bindings[:show] {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		s, p, o := termValue(bm, "s"), termValue(bm, "p"), termValue(bm, "o")
		fmt.Printf("  [%d] <%s>\n       <%s>\n       %q\n", i+1, s, p, o)
	}
	if len(bindings) > show {
		fmt.Printf("  … and %d more triple(s)\n", len(bindings)-show)
	}
}

// termValue safely extracts the "value" string from a SPARQL binding term.
func termValue(binding map[string]any, key string) string {
	if term, ok := binding[key].(map[string]any); ok {
		if v, ok := term["value"].(string); ok {
			return v
		}
	}
	return ""
}

// helper function to verify the catalog for a given beamline
func fabricCatalog(args []string, jsonOutput bool) {
	if len(args) < 2 {
		fmt.Println("ERROR: catalog requires <beamline>")
		fmt.Println("Usage: foxden fabric catalog <beamline>")
		os.Exit(1)
	}
	bl := args[1]

	rurl := fmt.Sprintf("%s/catalog/beamlines/%s/datasets",
		srvConfig.Config.Services.FabricCatalogURL, bl)
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}

	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("ERROR decoding response:", err)
		os.Exit(1)
	}

	if jsonOutput {
		if val, err := json.MarshalIndent(data, "", "  "); err == nil {
			fmt.Println(string(val))
		}
		return
	}

	// Human-readable catalog summary.
	dtype, _ := data["@type"].(string)
	fmt.Printf("Catalog for beamline: %s\n", bl)
	fmt.Printf("  @type: %s\n", dtype)

	datasets, _ := data["dcat:dataset"].([]any)
	fmt.Printf("  datasets listed: %d\n", len(datasets))

	if len(datasets) == 0 {
		fmt.Println("  WARNING: catalog lists 0 datasets for this beamline.")
		fmt.Println("  Possible cause: FOXDEN beamline field mismatch (array vs scalar, case mismatch).")
		return
	}

	for i, ds := range datasets {
		dsm, ok := ds.(map[string]any)
		if !ok {
			continue
		}
		id, _ := dsm["@id"].(string)
		title, _ := dsm["dct:title"].(string)
		accessURL := ""
		if dist, ok := dsm["dcat:distribution"].(map[string]any); ok {
			accessURL, _ = dist["dcat:accessURL"].(string)
		}
		fmt.Printf("  [%d] id=%s title=%q accessURL=%s\n", i+1, id, title, accessURL)
	}
}

// helper function to list content of a bucket on s3 storage
func fabricList(args []string, jsonOutput bool) {
	// args contains [ls beamline]
	if args[0] != "ls" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	// get beamlines datasets from fabric node
	bl := args[1]
	rurl := fmt.Sprintf("%s/catalog/beamlines/%s/datasets", srvConfig.Config.Services.FabricCatalogURL, bl)
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	if jsonOutput {
		if val, err := json.MarshalIndent(data, "", " "); err == nil {
			fmt.Println(string(val))
		}
		return
	}
	printMap(data)
}

// readDIDsFromFile reads one DID per non-blank, non-comment line from a file.
// Lines not starting with "/beamline=" are warned and skipped.
func readDIDsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var dids []string
	scanner := bufio.NewScanner(f)
	lineno := 0
	for scanner.Scan() {
		lineno++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "/beamline=") {
			fmt.Printf("WARNING: line %d: skipping malformed DID: %q\n", lineno, line)
			continue
		}
		dids = append(dids, line)
	}
	return dids, scanner.Err()
}

// ingestOneDID posts a single DID to the data-service ingest endpoint.
func ingestOneDID(did string) {
	bl := utils.GetBeamline(did)
	if bl == "" {
		fmt.Printf("ERROR: cannot extract beamline from DID %q\n", did)
		os.Exit(1)
	}
	encodedDid := url.PathEscape(did)
	rurl := fmt.Sprintf("%s/beamlines/%s/datasets/%s/foxden/ingest",
		srvConfig.Config.Services.FabricDataServiceURL, bl, encodedDid)
	if verbose > 0 {
		fmt.Println("HTTP POST", rurl)
	}
	resp, err := _httpWriteRequest.Post(rurl, "", bytes.NewBuffer([]byte{}))
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result IngestRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("ERROR decoding response:", err)
		os.Exit(1)
	}
	fmt.Printf("  ingested=%d did=%s graphIRI=%s\n", result.Ingested, result.Did, result.GraphIRI)
}

// helper function to ingest did (or a file of dids) into fabric node.
// The second argument is resolved in this order:
//  1. If the path exists on disk → treat as a file containing one DID per line.
//  2. Otherwise → treat as a literal DID string (must start with /beamline=).
func fabricIngest(args []string) {
	if len(args) != 2 {
		fmt.Println("ERROR: wrong number of arguments")
		fabricUsage()
		os.Exit(1)
	}
	if args[0] != "ingest" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}

	arg := args[1]

	// 1. Check whether the argument is an existing file first.
	if _, err := os.Stat(arg); err == nil {
		dids, err := readDIDsFromFile(arg)
		if err != nil {
			fmt.Println("ERROR reading DID file:", err)
			os.Exit(1)
		}
		if len(dids) == 0 {
			fmt.Println("ERROR: no valid DIDs found in file", arg)
			os.Exit(1)
		}
		fmt.Printf("Ingesting %d DID(s) from %s\n", len(dids), arg)
		for _, did := range dids {
			fmt.Printf("  → %s\n", did)
			ingestOneDID(did)
		}
		fmt.Printf("Done. %d DID(s) processed.\n", len(dids))
		return
	}

	// 2. Treat as a literal DID string.
	if !strings.HasPrefix(arg, "/beamline=") {
		fmt.Printf("ERROR: %q is neither an existing file nor a valid DID (must start with /beamline=)\n", arg)
		os.Exit(1)
	}
	fmt.Printf("Ingesting DID: %s\n", arg)
	ingestOneDID(arg)
}

func fabricCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fabric",
		Short: "foxden fabric commands",
		Long:  "foxden fabric commands to access CHESS FabricNode service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			limit, _ := cmd.Flags().GetInt("limit")
			if len(args) == 0 {
				fabricUsage()
				return
			}
			switch args[0] {
			case "ls":
				if len(args) < 2 {
					fmt.Println("ERROR: ls requires <beamline>")
					os.Exit(1)
				}
				accessToken()
				fabricList(args, jsonOutput)
			case "ingest":
				writeToken()
				fabricIngest(args)
			case "health":
				accessToken()
				fabricHealth(jsonOutput)
			case "sparql":
				accessToken()
				fabricSPARQL(args, jsonOutput, limit)
			case "catalog":
				accessToken()
				fabricCatalog(args, jsonOutput)
			default:
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
				fabricUsage()
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.PersistentFlags().Int("limit", 5, "number of SPARQL triples to display (0 = all)")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		fabricUsage()
		return nil
	})
	return cmd
}
