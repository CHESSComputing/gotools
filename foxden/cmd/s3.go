package cmd

// CHESComputing foxden tool: s3 module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	srvConfig "github.com/CHESSComputing/golib/config"
	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// StorageRecord represents Storage record returned by datamanagement service
type StorageRecord struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

// UploadRecord represents Storage record returned by datamanagement service
type UploadRecord struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Msg    string `json:"msg"`
	Object any    `json:"object"`
}

// helper function to provide s3 usage info
func s3Usage() {
	fmt.Println("foxden s3 <ls|create|delete|upload> [options]")
	fmt.Println("options: file or directory")
	fmt.Println("\nExamples:")
	fmt.Println("\n# create new bucket:")
	fmt.Println("foxden s3 create Cornell/bucket")
	fmt.Println("\n# remove bucket or file:")
	fmt.Println("foxden s3 delete Cornell/bucket")
	fmt.Println("\n# upload new file to a bucket:")
	fmt.Println("foxden s3 upload Cornell/bucket file.txt")
	fmt.Println("\n# upload all files from given directory to a bucket:")
	fmt.Println("foxden s3 upload Cornell/bucket someDirectory")
	fmt.Println("\n# list content of s3 storage:")
	fmt.Println("foxden s3 ls Cornell")
	fmt.Println("\n# list specific bucket on s3 storage:")
	fmt.Println("foxden s3 ls Cornell/bucket")
}

// helper function to list content of a bucket on s3 storage
func s3List(args []string) {
	// args contains [ls bucket]
	if args[0] != "ls" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	var bucketName string
	rurl := fmt.Sprintf("%s/storage", srvConfig.Config.Services.DataManagementURL)
	if len(args) > 1 {
		bucketName = args[1]
		rurl = fmt.Sprintf("%s/storage/%s", srvConfig.Config.Services.DataManagementURL, bucketName)
	}

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
	var results StorageRecord
	if err := dec.Decode(&results); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	data := results.Data
	switch v := data.(type) {
	case map[string]any:
		printMap(v)
	case []any:
		for _, rec := range v {
			switch vvv := rec.(type) {
			case map[string]any:
				printMap(vvv)
			default:
				fmt.Println("%+v %T", vvv, vvv)
			}
		}
	default:
		fmt.Println(v)
	}
	// fmt.Printf("results: %+v %T\n", results.Data, results.Data)
}

// helper function to print map entries
func printMap(m map[string]any) {
	maxLen := 20
	for k, v := range m {
		key := utils.PaddedKey(k, maxLen)
		switch vvv := v.(type) {
		case map[string]any:
			printMap(vvv)
		case []any:
			for _, rec := range vvv {
				switch vvv := rec.(type) {
				case string:
					fmt.Printf("%s: %s\n", key, vvv)
				case map[string]any:
					printMap(vvv)
				default:
					fmt.Printf("%s: %+v\n", key, vvv)
				}
			}
		case string:
			fmt.Printf("%s: %s\n", key, vvv)
		default:
			fmt.Printf("%s: %+v\n", key, vvv)
		}
	}
}

// helper function to create new bucket on s3 storage
func s3Create(args []string) {
	// args contains [create bucket]
	if len(args) != 2 {
		fmt.Println("ERROR: wrong number of arguments")
		os.Exit(1)
	}
	if args[0] != "create" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	bucketName := args[1]
	fmt.Printf("INFO: create bucket %s\n", bucketName)
	var results StorageRecord
	rurl := fmt.Sprintf("%s/storage/%s", srvConfig.Config.Services.DataManagementURL, bucketName)
	if verbose > 0 {
		fmt.Println("HTTP POST", rurl)
	}
	resp, err := _httpWriteRequest.Post(rurl, "", nil)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&results); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	fmt.Printf("results: %+v\n", results)
}

// isDirectory determines if a file represented
// by `path` is a directory or not
func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), err
}

// helper function to upload file or directory to bucket on s3 storage
func s3Upload(args []string) {
	// args contains [upload bucket file|dir]
	if len(args) != 3 {
		fmt.Println("ERROR: wrong number of arguments")
		os.Exit(1)
	}
	if args[0] != "upload" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	bucketName := args[1]
	fobj := args[2]
	var files []string
	isDir, err := isDirectory(fobj)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	if isDir {
		if dirFiles, err := ioutil.ReadDir(fobj); err == nil {
			for _, file := range dirFiles {
				if !file.IsDir() {
					files = append(files, file.Name())
				}
			}
		}
	} else {
		files = append(files, fobj)
	}
	for _, f := range files {
		fname := filepath.Join(fobj, f)
		fmt.Printf("INFO: upload %s to bucket %s\n", fname, bucketName)
		rurl := fmt.Sprintf("%s/storage/%s/%s", srvConfig.Config.Services.DataManagementURL, bucketName, f)
		if verbose > 0 {
			fmt.Println("HTTP POST", rurl)
		}
		// open file and read its content
		// TODO: we may need buffer stream to reduce RAM utilization
		file, err := os.Open(fname)
		if err != nil {
			fmt.Println("ERROR", err)
			os.Exit(1)
		}
		defer file.Close()

		// send POST request to DataManagement service with file data content
		// see https://stackoverflow.com/questions/20205796/post-data-using-the-content-type-multipart-form-data
		/*
		   ```
		    curl -X POST http://localhost:8340/storage/cornell/s3-bucket/archive.zip \
		     -F "file=@/path/test.zip" \
		     -H "Content-Type: multipart/form-data"
		   ```
		*/
		// prepare our payload by reading the local file and passing it to
		// multipart writer
		var buf bytes.Buffer
		var errBuf error
		w := multipart.NewWriter(&buf)
		if fw, err := w.CreateFormFile("file", file.Name()); err == nil {
			if _, err := io.Copy(fw, file); err != nil {
				errBuf = err
			}
		} else {
			errBuf = err
		}
		w.Close()
		if errBuf != nil {
			fmt.Println("ERROR:", errBuf)
			os.Exit(1)
		}

		req, err := http.NewRequest("POST", rurl, &buf)
		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		accessToken := os.Getenv("FOXDEN_WRITE_TOKEN")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		var results UploadRecord
		if err := dec.Decode(&results); err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
		fmt.Printf("results: %+v\n", results)
	}
}

// helper function to delete bucket on s3 storage
func s3Delete(args []string) {
	// args contains [delete bucket]
	if len(args) != 2 {
		fmt.Println("ERROR: wrong number of arguments")
		os.Exit(1)
	}
	if args[0] != "delete" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	bucketName := args[1]
	fmt.Printf("INFO: delete bucket %s\n", bucketName)
	var results StorageRecord
	rurl := fmt.Sprintf("%s/storage/%s", srvConfig.Config.Services.DataManagementURL, bucketName)
	if verbose > 0 {
		fmt.Println("HTTP DELETE", rurl)
	}
	req, err := http.NewRequest("DELETE", rurl, nil)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	accessToken := os.Getenv("FOXDEN_DELETE_TOKEN")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&results); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	fmt.Printf("results: %+v\n", results)
}

func s3Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3",
		Short: "foxden s3 commands",
		Long:  "foxden s3 commands to access FOXDEN DataManagement (S3) service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				s3Usage()
			} else if args[0] == "ls" {
				accessToken()
				s3List(args)
			} else if args[0] == "create" {
				writeToken()
				s3Create(args)
			} else if args[0] == "delete" {
				deleteToken()
				s3Delete(args)
			} else if args[0] == "upload" {
				writeToken()
				s3Upload(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		s3Usage()
		return nil
	})
	return cmd
}
