package cmd

// CHESComputing foxden tool: sync module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/spf13/cobra"
)

// helper function to provide usage of sync option
func syncUsage() {
	fmt.Println("foxden sync <service: meta or provenance> [options]")
	fmt.Println("options: --src=<src> --dst=<dst> --spec=<spec> --poolSize=<poolSize> --batchSize=<batchSize>")
	fmt.Println("\nExamples:")
	fmt.Println("\n# sync meta-data records:")
	fmt.Println("foxden sync meta --src=http://localhost:8300 --dst=https://foxden.... --spec={}")
}

// generic function to sync records from src to dst given spec (JSON query) and pool parameters
// this function makes the following assumptions
// the source URI presents records from /records end-point and support ndjson data-format
// the spec is a query in JSON format to fetch records
func syncRecords(src, dst, spec string, poolSize, batchSize int) {
	log.Println("not implemented yet")

	// fetch data from src uri
	data, err := json.Marshal(spec)
	rurl := fmt.Sprintf("%s/records", src)
	resp, err := _httpReadRequest.Request("GET", rurl, "application/x-ndjson", bytes.NewBuffer(data))
	if err != nil {
		msg := fmt.Sprintf("fail /records, unable to fetch data from service %s", src)
		exit(msg, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error fetching records: received status %s", resp.Status)
	}

	// Create a worker pool
	var wg sync.WaitGroup
	recordChan := make(chan map[string]interface{}, poolSize)

	// Start workers
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range recordChan {
				injectRecord(dst, record)
			}
		}()
	}
	// Read records in batches and send them to the worker pool
	decoder := json.NewDecoder(resp.Body)
	batch := make([]map[string]interface{}, 0, batchSize)
	for {
		var record map[string]interface{}
		if err := decoder.Decode(&record); err == io.EOF {
			// Handle any remaining records in the last batch
			for _, r := range batch {
				recordChan <- r
			}
			break
		} else if err != nil {
			log.Printf("Error decoding NDJSON data: %v", err)
			continue
		}

		batch = append(batch, record)
		if len(batch) == batchSize {
			for _, r := range batch {
				recordChan <- r
			}
			batch = batch[:0] // Clear the batch
		}
	}

	close(recordChan)
	wg.Wait()

	log.Println("Records synced successfully!")
}

// helper function to inject record into destination URI
func injectRecord(dst string, record map[string]interface{}) {
	data, err := json.Marshal(record)
	if err != nil {
		log.Printf("Error marshalling record: %v", err)
		return
	}

	rurl := fmt.Sprintf("%s", dst)
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		exit("error injecting record", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to inject record: received status %s", resp.Status)
	}
}

func syncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "foxden sync command",
		Long:  "foxden sync-data command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			spec, _ := cmd.Flags().GetString("spec")
			src, _ := cmd.Flags().GetString("src")
			dst, _ := cmd.Flags().GetString("dst")
			poolSize, _ := cmd.Flags().GetInt("poolSize")
			batchSize, _ := cmd.Flags().GetInt("batchSize")
			writeToken()
			if len(args) == 0 {
				syncUsage()
			} else if args[1] == "meta" || args[1] == "prov" {
				syncRecords(src, dst, spec, poolSize, batchSize)
			} else {
				syncUsage()
			}
		},
	}
	cmd.PersistentFlags().String("spec", "", "query spec (JSON)")
	cmd.PersistentFlags().String("src", "", "specify src uri")
	cmd.PersistentFlags().String("dst", "", "specify dst uri")
	cmd.PersistentFlags().Int("poolSize", 5, "pool size, default: 5")
	cmd.PersistentFlags().Int("batchSize", 10, "batch size, default: 10")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		syncUsage()
		return nil
	})
	return cmd
}
