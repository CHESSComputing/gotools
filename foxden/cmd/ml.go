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
	"time"

	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

// MLInput represents input for predict API
type MLInput struct {
	Model   string `json:"model"`
	Type    string `json:"type"`
	Backend string `json:"backend"`
	File    string `json:"file",omitempty`
	Version string `json:"version",omitempty`
}

// helper function to provide ml usage info
func mlUsage() {
	fmt.Println("foxden ml <models|upload|predict|delete> [options]")
	fmt.Println("options: --file=input-file --model=ml-model --type=ml-type --backend=ml-backend")
	fmt.Println("\nExamples:")
	fmt.Println("\n# upload new ML model:")
	fmt.Println("foxden ml upload --file=/path/file.tar.gz --model=model --type=TensorFlow --backend=GoFake")
	fmt.Println("\n# delete model:")
	fmt.Println("foxden ml delete --model=model --type=TensorFlow <--version=latest>")
	fmt.Println("\n# ML inference for input specified via JSON payload:")
	fmt.Println("foxden ml predict --file=/path/input.json")
	fmt.Println("\n# ML inference for input via submission, e.g. image prediction")
	fmt.Println("foxden ml predict --file=/path/img.png --model=model --type=TensorFlow --backend=TFaaS")
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
func mlPredict(rec MLInput) {
	// curl http://localhost:8350/predict -v -X POST -H "Authorization: bearer $token" -H "Accept: application/json" -H "Content-type: application/json" -d '{"input":[1,2,3], "model": "model", "type": "TensorFlow", "backend": "GoFake"}'

	rurl := fmt.Sprintf("%s/predict", _srvConfig.Services.MLHubURL)
	if verbose > 0 {
		fmt.Println("HTTP POST", rurl)
	}

	var resp *http.Response
	var err error
	if strings.HasSuffix(rec.File, "json") {
		// JSON input
		file, err := os.Open(rec.File)
		exit("fail to open file", err)
		defer file.Close()
		data, err := io.ReadAll(file)
		exit("fail to read file", err)

		resp, err = _httpReadRequest.Post(rurl, "application/json", bytes.NewBuffer(data))
		exit("fail to make HTTP request", err)
	} else {
		// Image input

		// new multipart writer.
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("model", rec.Model)
		writer.WriteField("type", rec.Type)
		writer.WriteField("backend", rec.Backend)
		arr := strings.Split(rec.File, "/")
		fieldName := arr[len(arr)-1]
		fw, err := writer.CreateFormFile("image", fieldName)
		exit("fail to create form file", err)
		file, err := os.Open(rec.File)
		exit("fail to open file", err)
		defer file.Close()
		_, err = io.Copy(fw, file)
		exit("fail to copy file", err)
		writer.Close()
		req, err := http.NewRequest("POST", rurl, bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", _httpReadRequest.Token))
		client := &http.Client{
			Timeout: time.Second * 10,
		}
		resp, err = client.Do(req)
		exit("fail to make HTTP request", err)
	}

	// TODO: properly parse arguments
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	exit("fail to read response", err)
	fmt.Println("MLHub response:")
	sdata := string(data)
	if err == nil {
		if strings.Contains(sdata, "error") {
			var response services.ServiceResponse
			if err := json.Unmarshal(data, &response); err == nil {
				fmt.Println("Http code     :", response.HttpCode)
				fmt.Println("Service code  :", response.SrvCode)
				fmt.Println("Service       :", response.Service)
				fmt.Println("Error         :", response.Error)
			} else {
				fmt.Println("Error         :", sdata)
			}
		} else {
			fmt.Println(sdata)
		}
	} else {
		fmt.Println("Error         :", err)
	}
}

// helper function to upload file or directory to bucket on ml storage
func mlUpload(rec MLInput) {
	// curl http://localhost:8350/upload -v -X POST -H "Authorization: bearer $t" -F 'file=@/Users/vk/Downloads/DataBookkeeping_Darwin_arm64.tar.gz' -F 'model=model' -F 'type=TensorFlow' -F 'backend=GoFake'

	fname := rec.File
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
	if rec.Model != "" {
		if err := w.WriteField("model", rec.Model); err != nil {
			formErrors = append(formErrors, err)
		}
	}
	if rec.Type != "" {
		if err := w.WriteField("type", rec.Type); err != nil {
			formErrors = append(formErrors, err)
		}
	}
	if rec.Backend != "" {
		if err := w.WriteField("backend", rec.Backend); err != nil {
			formErrors = append(formErrors, err)
		}
	}
	if rec.Version != "" {
		if err := w.WriteField("version", rec.Version); err != nil {
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
}

// helper function to delete bucket on ml storage
func mlDelete(rec MLInput) {
	// curl http://localhost:8350/delete -v -X DELETE -H "Authorization: bearer $token" -H "Accept: application/json" -H "Content-type: application/json" -d '{"model": "model", "type": "TensorFlow", "version": "latest"}'
	data, err := json.Marshal(rec)
	exit("fail to marshal parameters", err)

	fmt.Printf("INFO: delete %s\n", rec)

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
			mlModel, _ := cmd.Flags().GetString("model")
			mlType, _ := cmd.Flags().GetString("type")
			mlBackend, _ := cmd.Flags().GetString("backend")
			mlFile, _ := cmd.Flags().GetString("file")
			mlVersion, _ := cmd.Flags().GetString("version")
			rec := MLInput{
				Model: mlModel, Type: mlType, Backend: mlBackend, File: mlFile, Version: mlVersion,
			}
			if len(args) == 0 {
				mlUsage()
			} else if args[0] == "models" {
				accessToken()
				mlModels(args)
			} else if args[0] == "predict" {
				accessToken()
				mlPredict(rec)
			} else if args[0] == "delete" {
				deleteToken()
				mlDelete(rec)
			} else if args[0] == "upload" {
				writeToken()
				mlUpload(rec)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.PersistentFlags().String("model", "", "ML model name to use, e.g. mnist")
	cmd.PersistentFlags().String("type", "", "ML type, e.g. TensorFlow")
	cmd.PersistentFlags().String("file", "", "input file name, JSON or image")
	cmd.PersistentFlags().String("backend", "", "ML backend to use, e.g. TensorFlow")
	cmd.PersistentFlags().String("version", "", "ML model version")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		mlUsage()
		return nil
	})
	return cmd
}
