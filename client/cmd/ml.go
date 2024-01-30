package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// helper function to provide ml usage info
func mlUsage() {
	fmt.Println("client ml <models|upload|predict|delete> [values]")
	fmt.Println("Examples:")
	fmt.Println("\n# upload new ML model:")
	fmt.Println("client ml upload file=/path/file.tar.gz model=model type=TensorFlow backend=GoFake")
	fmt.Println("\n# delete model:")
	fmt.Println("client ml delete model=model type=TensorFlow")
	fmt.Println("\n# ML inference:")
	fmt.Println("client ml predict /path/input.json")
}

// helper function to list content of a bucket on ml storage
func mlModels(args []string) {
	// args contains [ls bucket]
	if args[0] != "ls" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	rurl := fmt.Sprintf("%s/models", _srvConfig.Services.MLHubURL)

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
	var results map[string]any
	if err := dec.Decode(&results); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	printMap(results)
}

// helper function to create new bucket on ml storage
func mlPredict(args []string) {
	// args contains [create bucket]
	if len(args) != 2 {
		fmt.Println("ERROR: wrong number of arguments")
		os.Exit(1)
	}
	if args[0] != "predict" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
}

// helper function to upload file or directory to bucket on ml storage
func mlUpload(args []string) {
	// get values for them from args
	var fname string
	params := make(map[string]string)
	for _, k := range args {
		if k == "upload" {
			continue
		}
		key := strings.Trim(k, " ")
		a := strings.Split(key, "=")
		if key == "file=" {
			fname = a[1]
		} else if key == "model=" {
			params["model"] = a[1]
		} else if key == "type=" {
			params["type"] = a[1]
		} else if key == "backend=" {
			params["backend"] = a[1]
		}

	}

	// curl http://localhost:8350/upload -v -X POST -H "Authorization: bearer $t" -F 'file=@/Users/vk/Downloads/DataBookkeeping_Darwin_arm64.tar.gz' -F 'model=model' -F 'type=TensorFlow' -F 'backend=GoFake'
	fmt.Printf("INFO: upload %s\n", fname)
	rurl := fmt.Sprintf("%s/upload", _srvConfig.Services.MLHubURL)
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

	// prepare our payload by reading the local file and passing it to
	// multipart writer
	var buf bytes.Buffer
	var formErrors []error
	w := multipart.NewWriter(&buf)
	if fw, err := w.CreateFormFile("file", file.Name()); err == nil {
		if _, err := io.Copy(fw, file); err != nil {
			formErrors = append(formErrors, err)
		}
	} else {
		formErrors = append(formErrors, err)
	}
	for key, val := range params {
		_, err := w.CreateFormFile(key, val)
		if err != nil {
			formErrors = append(formErrors, err)
		}
	}
	w.Close()
	for _, err := range formErrors {
		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}
	}

	req, err := http.NewRequest("POST", rurl, &buf)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	accessToken := os.Getenv("CHESS_WRITE_TOKEN")
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

// helper function to delete bucket on ml storage
func mlDelete(args []string) {
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
	rurl := fmt.Sprintf("%s/storage/%s", _srvConfig.Services.DataManagementURL, bucketName)
	if verbose > 0 {
		fmt.Println("HTTP DELETE", rurl)
	}
	req, err := http.NewRequest("DELETE", rurl, nil)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	accessToken := os.Getenv("CHESS_DELETE_TOKEN")
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

func mlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ml",
		Short: "client ml command",
		Long:  "client ml command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				mlUsage()
			} else if args[0] == "models" {
				accessToken()
				mlModels(args)
			} else if args[0] == "predict" {
				writeToken()
				mlPredict(args)
			} else if args[0] == "delete" {
				deleteToken()
				mlDelete(args)
			} else if args[0] == "upload" {
				writeToken()
				mlUpload(args)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		mlUsage()
		return nil
	})
	return cmd
}
