// foxden_injector.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type result struct {
	ElapsedMs float64
	ExitCode  int
	Did       string
	Index     int
}

func main() {
	var (
		filePath    string
		total       int
		concurrency int
		foxdenTool  string
		foxdenSrv   string
		foxdenCmd   string
		quiet       bool
		timeoutSec  int
	)
	flag.StringVar(&filePath, "file", "", "base JSON file to use (required)")
	flag.IntVar(&total, "n", 100, "total number of injections")
	flag.IntVar(&concurrency, "c", 10, "concurrency level")
	flag.StringVar(&foxdenTool, "foxdenTool", "", "path to foxden binary")
	flag.StringVar(&foxdenSrv, "foxdenSrv", "meta", "foxden service (meta, prov, sync, etc.)")
	flag.StringVar(&foxdenCmd, "foxdenCmd", "add", "foxden command for service (add, ls, view, delete, etc.)")
	flag.BoolVar(&quiet, "quiet", false, "suppress per-invocation prints")
	flag.IntVar(&timeoutSec, "timeout", 60, "per-invocation timeout seconds (0 = no timeout)")
	flag.Parse()

	if filePath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -file meta.json [-n total] [-c concurrency] [-foxden ./foxden]\n", os.Args[0])
		os.Exit(2)
	}

	// Read and parse JSON
	baseBytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading JSON file: %v\n", err)
		os.Exit(1)
	}
	var base map[string]interface{}
	if err := json.Unmarshal(baseBytes, &base); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid JSON: %v\n", err)
		os.Exit(1)
	}

	didBase, ok := base["did"].(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: top-level 'did' key not found or not a string\n")
		os.Exit(1)
	}

	if _, err := os.Stat(foxdenTool); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: foxden binary not found at %s\n", foxdenTool)
	}

	if !quiet {
		fmt.Printf("Base JSON: %s\n", filePath)
		fmt.Printf("foxden path: %s\n", foxdenTool)
		fmt.Printf("Total: %d  Concurrency: %d\n", total, concurrency)
	}

	// Concurrency control
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	results := make([]result, 0, total)
	var resultsMu sync.Mutex

	var succCount, failCount int64
	var counter uint64
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 1; i <= total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			// Create unique did
			c := atomic.AddUint64(&counter, 1)
			newDid := fmt.Sprintf("%s-%d-%d-%04d", didBase, time.Now().UnixNano(), c, randSrc.Intn(10000))

			// Prepare JSON payload
			m := make(map[string]interface{}, len(base))
			for k, v := range base {
				m[k] = v
			}
			m["did"] = newDid
			body, _ := json.Marshal(m)

			// Build command
			ctx := context.Background()
			if timeoutSec > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
				defer cancel()
			}
			cmd := exec.CommandContext(ctx, foxdenTool, foxdenSrv, foxdenCmd, "-")
			if foxdenCmd != "add" || foxdenCmd != "amend" {
				cmd = exec.CommandContext(ctx, foxdenTool, foxdenSrv, foxdenCmd)
			}
			cmd.Stdin = bytes.NewReader(body)

			start := time.Now()
			err := cmd.Run()
			elapsed := time.Since(start)
			ms := float64(elapsed.Nanoseconds()) / 1e6

			exitCode := 0
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					exitCode = ee.ExitCode()
				} else if err == context.DeadlineExceeded {
					exitCode = 124
				} else {
					exitCode = 1
				}
			}

			if exitCode == 0 {
				atomic.AddInt64(&succCount, 1)
				if !quiet {
					fmt.Printf("[%d] OK did=%s time=%.2fms\n", idx, newDid, ms)
				}
			} else {
				atomic.AddInt64(&failCount, 1)
				fmt.Fprintf(os.Stderr, "[%d] FAIL did=%s code=%d time=%.2fms\n", idx, newDid, exitCode, ms)
			}

			resultsMu.Lock()
			results = append(results, result{ElapsedMs: ms, ExitCode: exitCode, Did: newDid, Index: idx})
			resultsMu.Unlock()
		}(i)
	}

	wg.Wait()

	printStats(results, succCount, failCount, total)
}

func printStats(results []result, succCount, failCount int64, total int) {
	fmt.Println()
	fmt.Printf("Total attempted: %d\n", total)
	fmt.Printf("Successes:       %d\n", succCount)
	fmt.Printf("Failures:        %d\n", failCount)

	var times []float64
	for _, r := range results {
		if r.ExitCode == 0 {
			times = append(times, r.ElapsedMs)
		}
	}

	if len(times) == 0 {
		fmt.Println("No successful injections.")
		return
	}
	sort.Float64s(times)

	sum := 0.0
	for _, t := range times {
		sum += t
	}
	fmt.Println()
	fmt.Printf("Durations (ms):\n")
	fmt.Printf("  count: %d\n", len(times))
	fmt.Printf("  avg:   %.2f\n", sum/float64(len(times)))
	fmt.Printf("  min:   %.2f\n", times[0])
	fmt.Printf("  p50:   %.2f\n", percentile(times, 50))
	fmt.Printf("  p90:   %.2f\n", percentile(times, 90))
	fmt.Printf("  p99:   %.2f\n", percentile(times, 99))
	fmt.Printf("  max:   %.2f\n", times[len(times)-1])
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	pos := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	frac := pos - float64(lower)
	return sorted[lower] + frac*(sorted[upper]-sorted[lower])
}
