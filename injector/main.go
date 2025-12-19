package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

/* =========================
   Config & Flags
========================= */

type Config struct {
	Path    string
	Pattern string
	Token   string
	Schema  string
	Inject  bool
	URL     string
	Workers int
	Output  string
	Timeout time.Duration
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Path, "path", "", "root directory to crawl")
	flag.StringVar(&cfg.Pattern, "file", "", "file glob pattern (e.g. meta*.json)")
	flag.BoolVar(&cfg.Inject, "inject", false, "enable HTTP injection")
	flag.StringVar(&cfg.Token, "token", "", "FOXDEN write token")
	flag.StringVar(&cfg.Schema, "schema", "", "FOXDEN schema name")
	flag.StringVar(&cfg.URL, "url", "", "HTTP POST endpoint")
	flag.IntVar(&cfg.Workers, "workers", 4, "number of concurrent workers")
	flag.StringVar(&cfg.Output, "output", "", "log file")
	flag.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "HTTP timeout")

	flag.Parse()

	if cfg.Path == "" || cfg.Pattern == "" {
		flag.Usage()
		os.Exit(1)
	}
	if cfg.Inject && cfg.URL == "" {
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

/* =========================
   File Discovery
========================= */

func findFiles(root, pattern string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		match, _ := filepath.Match(pattern, filepath.Base(path))
		if match {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

/* =========================
   Token functions
========================= */

func getToken(token string) (string, error) {
	// 1. Fallback to env
	if token == "" {
		token = os.Getenv("FOXDEN_WRITE_TOKEN")
	}

	if token == "" {
		return "", fmt.Errorf("no token provided via flag or FOXDEN_WRITE_TOKEN")
	}

	// 2. Check if token is a file path
	info, err := os.Stat(token)
	if err == nil && !info.IsDir() {
		data, err := os.ReadFile(token)
		if err != nil {
			return "", fmt.Errorf("failed to read token file: %w", err)
		}
		token = string(data)
	}

	token = strings.TrimSpace(token)

	// 3. Validate
	if err := validateToken(token); err != nil {
		return "", err
	}

	return token, nil
}

func validateToken(token string) error {
	if token == "" {
		return fmt.Errorf("token is empty")
	}

	if strings.ContainsAny(token, " \t\r\n") {
		return fmt.Errorf("token contains whitespace or newlines")
	}

	if len(token) < 8 {
		return fmt.Errorf("invalid token: it is too short")
	}

	return nil
}

/* =========================
   Injection
========================= */

type InjectResult struct {
	Status int
	Body   string
	File   string
	Error  string
}

type FoxdenResponse struct {
	Error string
}

type MetadataRecord struct {
	Schema string
	Record map[string]any
}

func injectJSON(ctx context.Context,
	client *http.Client,
	url, file, schema, token string) (*InjectResult, error) {

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rec map[string]any
	err = json.Unmarshal(data, &rec)
	if err != nil {
		return nil, err
	}
	if schema == "" {
		if val, ok := rec["schema"]; ok {
			schema = fmt.Sprintf("%v", val)
		}
	}
	if schema == "" {
		return nil, errors.New("metadata record without schema name")
	}
	mrec := MetadataRecord{Schema: schema, Record: rec}
	data, err = json.Marshal(mrec)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var errStr string
	if err == nil {
		var rec FoxdenResponse
		if e := json.Unmarshal(body, &rec); e == nil {
			errStr = rec.Error
		}
	} else {
		errStr = err.Error()
	}

	return &InjectResult{
		Status: resp.StatusCode,
		Body:   string(body),
		File:   file,
		Error:  errStr,
	}, nil
}

/* =========================
   Workers & Crawler
========================= */

func worker(jobs <-chan string, results chan<- *InjectResult, wg *sync.WaitGroup, cfg *Config, client *http.Client) {
	defer wg.Done()

	for file := range jobs {
		if !cfg.Inject {
			results <- &InjectResult{
				Status: 0,
				Body:   "dry-run, no injection step is performed",
				File:   file,
			}
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		res, err := injectJSON(ctx, client, cfg.URL, file, cfg.Schema, cfg.Token)
		cancel()

		if err != nil {
			results <- &InjectResult{
				Status: 0,
				Body:   err.Error(),
				File:   file,
			}
			continue
		}

		results <- res
	}
}

func crawlAndInject(cfg *Config, files []string) error {
	jobs := make(chan string, cfg.Workers*2)
	results := make(chan *InjectResult, cfg.Workers)

	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go worker(jobs, results, &wg, cfg, client)
	}

	go func() {
		for _, f := range files {
			jobs <- f
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var logFile *os.File
	var err error
	if cfg.Output != "" {
		logFile, err = os.OpenFile(cfg.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer logFile.Close()
	}

	for res := range results {
		line := fmt.Sprintf("%d\t%s\t%s\n", res.Status, res.File, res.Error)
		if logFile != nil {
			logFile.WriteString(line)
		} else {
			fmt.Print(line)
		}
	}

	return nil
}

/* =========================
   Main
========================= */

func main() {
	cfg := parseFlags()

	token, err := getToken(cfg.Token)
	if err != nil {
		panic(err)
	}
	cfg.Token = token

	files, err := findFiles(cfg.Path, cfg.Pattern)
	if err != nil {
		panic(err)
	}

	if len(files) == 0 {
		fmt.Println("No files found")
		return
	}

	if err := crawlAndInject(cfg, files); err != nil {
		panic(err)
	}
}
