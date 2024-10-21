package cmd

// CHESComputing foxden tool: material commons module
//
// Copyright (c) 2024 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	mcapi "github.com/materials-commons/gomcapi"
	"github.com/materials-commons/hydra/pkg/mcdb/mcmodel"
	"github.com/spf13/cobra"
)

// helper function to get metadata
// MaterialCommons represents MaterialCommons object returned from discovery service
type MaterialCommons struct {
	ID          string   `json:"id"`
	Site        string   `json:"site"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Bucket      string   `json:"bucket"`
}

// MetaDataRecord represents MetaData record returned by discovery service
type MCDataRecord struct {
	Status string            `json:"status"`
	Data   []MaterialCommons `json:"data"`
}

// MCResponse represents HTTP response from Material Commons API
type MCResponse struct {
	Data []map[string]any `json:"data"`
}

var mcClient *mcapi.Client

func getMcProjectName() string {
	name := os.Getenv("FOXDEN_DOI_PROJECT")
	if name == "" {
		if _srvConfig.DOI.ProjectName != "" {
			name = _srvConfig.DOI.ProjectName
		} else {
			name = "FOXDEN"
		}
	}
	return name
}

// helper function to get project Id from a project name
func getMcProjectId() int {
	name := getMcProjectName()
	records, err := mcClient.ListProjects()
	exit("unable to list projects", err)
	for _, r := range records {
		if r.Name == name {
			return r.ID
		}
	}
	exit(fmt.Sprintf("unable to find project %s", name), errors.New("unknown project"))
	// we should not reach this ever
	return 0
}
func getMcClient() {
	if mcClient != nil {
		return
	}
	args := &mcapi.ClientArgs{
		BaseURL: _srvConfig.DOI.URL,
		APIKey:  _srvConfig.DOI.AccessToken,
	}
	mcClient = mcapi.NewClient(args)
	return
}

func mcView(pid int64) {
	mcListDatasets(int(pid))
}

func mcUpdate(did int64, fname string) {
}

func mcDocs(did int64) {
	mcListProjects()
}
func mcAdd(did int64, fname string) {
	if fname == "" {
		exit("not input file is provided", errors.New("missing file"))
	}
	// look-up project name from given dataset ID
	pid := getMcProjectId()

	// find our dataset with given did
	dataset := findMcDataset(pid, int(did))

	// create directory using dataset UUID
	dir, err := mcClient.CreateDirectoryByPath(pid, "/"+dataset.UUID)
	exit("unable to create dataset directory", err)

	// upload file to dataset directory
	_, err = mcClient.UploadFileTo(pid, fname, dir.Path)
	exit("unable to upload file to dataset directory", err)
	fmt.Printf("SUCCESS: a file %s has been added to dataset %s within project %s\n", fname, dataset.Name, getMcProjectName())
}

func createMcDataset(pid int, name, description, summary string) {
	req := mcapi.CreateOrUpdateDatasetRequest{
		Name:        name,
		Description: description,
		Summary:     summary,
	}
	ds, err := mcClient.CreateDataset(pid, req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New dataset has been created...\n")
	fmt.Printf("ID       : %+v\n", ds.ID)
	fmt.Printf("UUID     : %+v\n", ds.UUID)
	fmt.Printf("Name     : %+v\n", ds.Name)
	fmt.Printf("DOI      : %+v\n", ds.DOI)
	fmt.Printf("Created  : %+v\n", ds.CreatedAt)
	fmt.Printf("Published: %+v\n", ds.PublishedAt)
}

func uploadMcFile(pid int, fname string) {
	dirId := 0 // to be defined somehow
	fs, err := mcClient.UploadFile(pid, dirId, fname)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New file has been uploaded...\n")
	fmt.Printf("ID       : %+v\n", fs.ID)
	fmt.Printf("UUID     : %+v\n", fs.UUID)
	fmt.Printf("Name     : %+v\n", fs.Name)
	fmt.Printf("Path     : %+v\n", fs.Path)
	fmt.Printf("Created  : %+v\n", fs.CreatedAt)
}

func publishMcDataset(pid, mcDid int) {
	_, err := mcClient.PublishDataset(pid, mcDid)
	exit("unable to publish dataset", err)
	ds, err := mcClient.MintDOIForDataset(pid, mcDid)
	exit("unable to mint DOI for dataset", err)
	fmt.Printf("Dataset has been published...\n")
	fmt.Printf("ID       : %+v\n", ds.ID)
	fmt.Printf("UUID     : %+v\n", ds.UUID)
	fmt.Printf("Name     : %+v\n", ds.Name)
	fmt.Printf("DOI      : %+v\n", ds.DOI)
	fmt.Printf("Created  : %+v\n", ds.CreatedAt)
	fmt.Printf("Published: %+v\n", ds.PublishedAt)
}

func getMcProject(pid int) {
	proj, err := mcClient.GetProject(pid)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ID       : %+v\n", proj.ID)
	fmt.Printf("UUID     : %s\n", proj.UUID)
	fmt.Printf("Name     : %s\n", proj.Name)
	fmt.Printf("Owner    : %s (%s)\n", proj.Owner.Name, proj.Owner.Email)
	fmt.Printf("Size     : %+v\n", proj.Size)
	fmt.Printf("FileCount: %+v\n", proj.FileCount)
	fmt.Printf("Created  : %+v\n", proj.CreatedAt)
	fmt.Printf("Updated  : %+v\n", proj.UpdatedAt)
}

// helper function to get meta-data records
func getMaterialCommons(user, query string) ([]map[string]any, error) {
	var records []map[string]any
	materialCommonsUrl := "https://materialscommons.org/api"
	if _srvConfig.DOI.URL != "" {
		materialCommonsUrl = _srvConfig.DOI.URL
	}
	rurl := fmt.Sprintf("%s/projects", materialCommonsUrl)
	_httpReadRequest.Token = _srvConfig.DOI.AccessToken
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from meta-data service", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("data:", string(data))
		exit("unable to read data from meta-data service", err)
	}
	var mcRecord MCResponse
	err = json.Unmarshal(data, &mcRecord)

	if err != nil {
		log.Println("response data", string(data))
		exit("Unable to unmarshal the data", err)
	}
	for _, rec := range mcRecord.Data {
		records = append(records, rec)
	}
	return records, nil
}

// helper function to provide usage of meta option
func mcUsage() {
	fmt.Println("foxden mc <ls|rm|view> [options]")
	fmt.Println("foxden mc add <file.json> {options}")
	fmt.Println("options: --schema=<schema> --did-attrs=<attrs> --did-sep=<separator> --did-div=<divider> --json")
	fmt.Println("\nExamples:")
	fmt.Println("\n# create new document from given record:")
	fmt.Println("foxden mc create </path/record.json>")
	fmt.Println("\n# list all mc data records:")
	fmt.Println("foxden mc ls")
	fmt.Println("\n# view project datasets:")
	fmt.Println("foxden mc view <project-id>")
}

// helper function to list MaterialCommons projects
func mcListProjects() {
	records, err := mcClient.ListProjects()
	exit("unable to list projects", err)
	for _, r := range records {
		fmt.Printf("ID         : %+v\n", r.ID)
		fmt.Printf("Name       : %+v\n", r.Name)
		fmt.Printf("Description: %+v\n", r.Description)
		fmt.Printf("Summary    : %+v\n", r.Summary)
	}
	fmt.Println("---")
	fmt.Printf("Total      : %d records\n", len(records))
}

// helper function to list MaterialCommons datasets within given project id
func mcListDatasets(projID int) {
	records, err := mcClient.ListDatasets(projID)
	exit("unable to list datasets", err)
	for _, r := range records {
		fmt.Printf("ID         : %+v\n", r.ID)
		fmt.Printf("Name       : %+v\n", r.Name)
		fmt.Printf("Description: %+v\n", r.Description)
		fmt.Printf("Summary    : %+v\n", r.Summary)
		fmt.Printf("Authors    : %+v\n", r.Authors)
		fmt.Printf("DOI        : %+v\n", r.DOI)
		fmt.Printf("CreatedAt  : %+v\n", r.CreatedAt)
		fmt.Printf("Files      :\n")
		for _, f := range r.Files {
			fmt.Printf("ID       : %+v\n", f.CreatedAt)
			fmt.Printf("Name     : %+v\n", f.CreatedAt)
			fmt.Printf("Path     : %+v\n", f.CreatedAt)
			fmt.Printf("Size     : %+v\n", f.CreatedAt)
			fmt.Printf("MimeType : %+v\n", f.CreatedAt)
			fmt.Printf("CreatedAt: %+v\n", f.CreatedAt)
		}
		fmt.Println("---")
	}
	fmt.Printf("Total      : %d records\n", len(records))
}

// helper function to find MaterialCommons dataset name for given dataset id
func findMcDataset(pid, did int) *mcmodel.Dataset {
	records, err := mcClient.ListDatasets(pid)
	exit("unable to list datasets", err)
	for _, r := range records {
		if r.ID == did {
			return &r
		}
	}
	return nil
}

// helper function to list meta-data records
func mcListRecord(user, spec string, jsonOutput bool) {
	records, err := getMaterialCommons(user, spec)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	if jsonOutput {
		if data, err := json.MarshalIndent(records, "", " "); err == nil {
			fmt.Println(string(data))
		} else {
			fmt.Println("ERROR", err)
			os.Exit(1)
		}
		return
	}

	for _, r := range records {
		fmt.Printf("%+v", r)
	}
	fmt.Println("---")
	fmt.Println("Total   :", len(records), "records")

}

// helper function to create new project ID
func mcCreate(fname string) {
	tstamp := time.Now().String()
	name := fmt.Sprintf("FOXDEN dataset placeholder: %s", tstamp)
	description := fmt.Sprintf("FOXDEN dataset description: %s", tstamp)
	summary := fmt.Sprintf("FOXDEN dataset summary: %s", tstamp)
	deposit := mcapi.DepositDatasetRequest{
		Metadata: mcapi.DatasetMetadata{
			Name:        name,
			Description: description,
			Summary:     summary,
		},
	}

	if fname != "" {
		file, err := os.Open(fname)
		exit("unable to open file", err)
		defer file.Close()
		data, err := io.ReadAll(file)
		exit("unable to read file", err)
		err = json.Unmarshal(data, &deposit)
		name = deposit.Metadata.Name
		description = deposit.Metadata.Description
		summary = deposit.Metadata.Summary
	}

	// look-up FOXDEN project in MaterialCommons
	pid := getMcProjectId()

	ds, err := mcClient.DepositDataset(pid, deposit)
	exit("unable to deposit data to MaterialCommons", err)
	fmt.Printf("SUCCESS  : new deposit has been made to:\n")
	fmt.Printf("Name     : %s\n", getMcProjectName())
	fmt.Printf("ProjectID: %v\n", pid)
	fmt.Printf("Dataset  : %s\n", name)
	fmt.Printf("DatasetID: %v\n", ds.ID)
}

func materialCommonsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mc",
		Short: "foxden MaterialCommons commands",
		Long:  "foxden MaterialCommons commands to access FOXDEN MaterialCommons service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				// set _jsonOutputError to properly handle error output in JSON format
				_jsonOutputError = true
			}
			if len(args) == 0 {
				mcUsage()
				return
			}
			getMcClient()
			if args[0] == "add" {
				if pid, err := strconv.Atoi(args[1]); err == nil {
					if len(args) > 1 {
						did := args[2]
						description := "foxden dataset"
						summary := "foxden dataset summary"
						createMcDataset(pid, did, description, summary)
					}
				}
			} else if args[0] == "create" {
				mcCreate(args[1])
			} else if args[0] == "view" {
				if len(args) == 2 {
					if pid, err := strconv.Atoi(args[1]); err == nil {
						mcListDatasets(pid)
					}
				}
			} else if args[0] == "ls" {
				//                 user, _ := getUserToken()
				if len(args) == 2 {
					if pid, err := strconv.Atoi(args[1]); err == nil {
						getMcProject(pid)
					}
				} else {
					//                     mcListRecord(user, "", jsonOutput)
					mcListProjects()
				}
			} else if args[0] == "upload" {
				if len(args) == 3 {
					if pid, err := strconv.Atoi(args[1]); err == nil {
						uploadMcFile(pid, args[2])
					}
				}
			} else if args[0] == "publish" {
				if len(args) != 3 {
					log.Fatal("unable to publish MaterialCommons dataset, please provide project and dataset IDs")
				}
				if pid, err := strconv.Atoi(args[1]); err == nil {
					if mcDid, err := strconv.Atoi(args[2]); err == nil {
						publishMcDataset(pid, mcDid)
					} else {
						fmt.Println("WARNING: please provide MaterialCommons dataset ID")
					}
				} else {
					fmt.Println("WARNING: please provide project ID")
				}
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v", args)
			}
		},
	}
	cmd.PersistentFlags().Bool("json", false, "json output")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		mcUsage()
		return nil
	})
	return cmd
}
