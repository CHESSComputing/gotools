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
	fmt.Println("foxden ml <models|upload|predict|delete> [values]")
	fmt.Println("Examples:")
	fmt.Println("\n# upload new ML model:")
	fmt.Println("foxden ml upload file=/path/file.tar.gz model=model type=TensorFlow backend=GoFake")
	fmt.Println("\n# delete model:")
	fmt.Println("foxden ml delete model=model type=TensorFlow <version=latest>")
	fmt.Println("\n# ML inference:")
	fmt.Println("foxden ml predict /path/input.json")
}

// helper function to get ML data from MLHub
func mlGet(endpoint string, args []string) {
	rurl := fmt.Sprintf("%s/%s", _srvConfig.Services.MLHubURL, endpoint)
	if verbose > 0 {
		fmt.Println("HTTP GET", rurl)
	}
	resp, err := _httpReadRequest.Get(rurl)
	exit("unable to make HTTP request", err)
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var results []map[string]any
	err = dec.Decode(&results)
	exit("unable to decode results", err)
	for _, rec := range results {
		printMap(rec)
	}
}

// helper function to list content of a bucket on ml storage
func mlModels(args []string) {
	if args[0] != "models" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	// curl http://localhost:8350/models
	mlGet("models", args)
}

// helper function to create new bucket on ml storage
func mlPredict(args []string) {
	// curl http://localhost:8350/predict -v -X POST -H "Authorization: bearer $token" -H "Accept: application/json" -H "Content-type: application/json" -d '{"input":[1,2,3], "model": "model", "type": "TensorFlow", "backend": "GoFake"}'
	if args[0] != "predict" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	fname := args[1]
	file, err := os.Open(fname)
	exit("fail to open file", err)
	defer file.Close()
	data, err := io.ReadAll(file)
	exit("fail to read file", err)

	rurl := fmt.Sprintf("%s/predict", _srvConfig.Services.MLHubURL)
	if verbose > 0 {
		fmt.Println("HTTP POST", rurl)
	}
	resp, err := _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
	exit("fail to make HTTP request", err)
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	exit("fail to read response", err)
	fmt.Println("MLHub response:")
	fmt.Println(string(data))

	// dec := json.NewDecoder(resp.Body)
	// var results map[string]any
	//
	//	if err := dec.Decode(&results); err != nil {
	//	    fmt.Println("ERROR:", err)
	//	    os.Exit(1)
	//	}
	//
	// printMap(results)
}

// helper function to upload file or directory to bucket on ml storage
func mlUpload(args []string) {
	// curl http://localhost:8350/upload -v -X POST -H "Authorization: bearer $t" -F 'file=@/Users/vk/Downloads/DataBookkeeping_Darwin_arm64.tar.gz' -F 'model=model' -F 'type=TensorFlow' -F 'backend=GoFake'
	// get values for them from args
	var fname string
	params := make(map[string]string)
	for _, k := range args {
		if k == "upload" {
			continue
		}
		key := strings.Trim(k, " ")
		a := strings.Split(key, "=")
		if a[0] == "file" {
			fname = a[1]
		} else {
			params[a[0]] = a[1]
		}
	}
	fmt.Printf("INFO: upload %s\n", fname)
	rurl := fmt.Sprintf("%s/upload", _srvConfig.Services.MLHubURL)
	if verbose > 0 {
		fmt.Println("HTTP POST", rurl)
	}
	// open file and read its content
	// TODO: we may need buffer stream to reduce RAM utilization
	file, err := os.Open(fname)
	exit("fail to open file", err)
	defer file.Close()

	// prepare our payload by reading the local file and passing it to
	// multipart writer
	var buf bytes.Buffer
	var formErrors []error
	w := multipart.NewWriter(&buf)
	// add file form key and copy file content to the form
	if fw, err := w.CreateFormFile("file", file.Name()); err == nil {
		if _, err := io.Copy(fw, file); err != nil {
			formErrors = append(formErrors, err)
		}
	} else {
		formErrors = append(formErrors, err)
	}
	// add form field key=value pairs
	for key, val := range params {
		if err := w.WriteField(key, val); err != nil {
			formErrors = append(formErrors, err)
		}
	}
	w.Close()
	for _, err := range formErrors {
		exit("form error", err)
	}

	req, err := http.NewRequest("POST", rurl, &buf)
	exit("fail to make HTTP request", err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	accessToken := os.Getenv("CHESS_WRITE_TOKEN")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := http.Client{}
	resp, err := client.Do(req)
	exit("fail to make HTTP request", err)
	fmt.Println("MLHub response:", resp.Status)
	// defer resp.Body.Close()
	// dec := json.NewDecoder(resp.Body)
	// var results UploadRecord
	//
	//	if err := dec.Decode(&results); err != nil {
	//	    fmt.Println("ERROR:", err)
	//	    os.Exit(1)
	//	}
	//
	// fmt.Printf("results: %+v\n", results)
}

// helper function to delete bucket on ml storage
func mlDelete(args []string) {
	// curl http://localhost:8350/delete -v -X DELETE -H "Authorization: bearer $token" -H "Accept: application/json" -H "Content-type: application/json" -d '{"model": "model", "type": "TensorFlow", "version": "latest"}'
	if args[0] != "delete" {
		fmt.Println("ERROR: wrong action", args)
		os.Exit(1)
	}
	params := make(map[string]string)
	for _, k := range args {
		if k == "delete" {
			continue
		}
		key := strings.Trim(k, " ")
		a := strings.Split(key, "=")
		params[a[0]] = a[1]
	}
	if _, ok := params["version"]; !ok {
		params["version"] = "latest"
	}
	data, err := json.Marshal(params)
	exit("fail to marshal parameters", err)

	fmt.Printf("INFO: delete %s\n", params)

	rurl := fmt.Sprintf("%s/delete", _srvConfig.Services.MLHubURL)
	if verbose > 0 {
		fmt.Println("HTTP DELETE", rurl)
	}
	resp, err := _httpDeleteRequest.Delete(rurl, "application/json", bytes.NewBuffer(data))
	exit("fail to make HTTP request", err)
	fmt.Println("MLHub response:", resp.Status)
}

func mlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ml",
		Short: "foxden ml commands",
		Long:  "foxden ml commands to access FOXDEN MLHub service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				mlUsage()
			} else if args[0] == "models" {
				accessToken()
				mlModels(args)
			} else if args[0] == "predict" {
				accessToken()
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
