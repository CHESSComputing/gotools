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

	mcapi "github.com/materials-commons/gomcapi"
	"github.com/materials-commons/hydra/pkg/mcdb/mcmodel"
)

// helper function to get metadata
// MaterialsCommons represents MaterialCommons object returned from discovery service
type MaterialsCommons struct {
	ID          string   `json:"id"`
	Site        string   `json:"site"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Bucket      string   `json:"bucket"`
}

// MetaDataRecord represents MetaData record returned by discovery service
type MCDataRecord struct {
	Status string             `json:"status"`
	Data   []MaterialsCommons `json:"data"`
}

// MCResponse represents HTTP response from Material Commons API
type MCResponse struct {
	Data []map[string]any `json:"data"`
}

var mcClient *mcapi.Client

func getMcProjectName() string {
	name := os.Getenv("FOXDEN_DOI_PROJECT")
	if name == "" {
		if _srvConfig.DOI.MaterialsCommons.ProjectName != "" {
			name = _srvConfig.DOI.MaterialsCommons.ProjectName
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

	// if project does not exist we'll create it
	req := mcapi.CreateProjectRequest{
		Name:        name,
		Description: "FOXDEN project description",
		Summary:     "FOXDEN project summary",
	}
	proj, err := mcClient.CreateProject(req)
	exit(fmt.Sprintf("unable to create project %s", name), err)
	fmt.Printf("SUCCESS: created new project '%s' in MaterialsCommons\n\n", name)
	return proj.ID
}

// helper function to get MaterialsCommons client
func getMcClient() {
	if mcClient != nil {
		return
	}
	args := &mcapi.ClientArgs{
		BaseURL: _srvConfig.DOI.MaterialsCommons.Url,
		APIKey:  _srvConfig.DOI.MaterialsCommons.AccessToken,
	}
	mcClient = mcapi.NewClient(args)
	return
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

// helper function to list MaterialsCommons projects
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

// helper function to list MaterialsCommons datasets within given project id
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

// helper function to find MaterialsCommons dataset name for given dataset id
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

// helper function to create new project ID
func mcCreate(fname string) {
	name := "FOXDEN dataset placeholder"
	description := "FOXDEN dataset description"
	summary := "FOXDEN dataset summary"
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

	// look-up FOXDEN project in MaterialsCommons
	pid := getMcProjectId()

	ds, err := mcClient.DepositDataset(pid, deposit)
	exit("unable to deposit data to MaterialsCommons", err)
	fmt.Printf("SUCCESS  : new deposit has been made to:\n")
	fmt.Printf("Name     : %s\n", getMcProjectName())
	fmt.Printf("ProjectID: %v\n", pid)
	fmt.Printf("Dataset  : %s\n", name)
	fmt.Printf("DatasetID: %v\n", ds.ID)
}

// view details of all dataset within given project
func mcView(pid int64) {
	mcListDatasets(int(pid))
}

// helper function to update meta-data record in MaterialsCommons
func mcUpdate(did int64, fname string) {
	if fname != "" {
		exit("no given file name", errors.New("unknown file"))
	}
	// find project id for
	pid := getMcProjectId()

	// load meta-data record from given file to create deposit
	var deposit mcapi.DepositDatasetRequest
	file, err := os.Open(fname)
	exit("unable to open file", err)
	defer file.Close()
	data, err := io.ReadAll(file)
	exit("unable to read file", err)
	err = json.Unmarshal(data, &deposit)
	exit("unable to unmarshal the data into deposit", err)

	// make update request with our new meta-data
	req := mcapi.CreateOrUpdateDatasetRequest{
		Name:        deposit.Metadata.Name,
		Description: deposit.Metadata.Description,
		Summary:     deposit.Metadata.Summary,
		License:     deposit.Metadata.License,
		Funding:     deposit.Metadata.Funding,
		Communities: deposit.Metadata.Communities,
		Authors:     deposit.Metadata.Authors,
		Tags:        deposit.Metadata.Tags,
	}

	ds, err := mcClient.CreateDataset(pid, req)
	exit("unable to update dataset", err)
	fmt.Printf("SUCCESS: updated project %s with new dataset meta-data %+v\n", getMcProjectName(), ds)
}

// helper function to list all projects within Material Commons
func mcDocs(did int64) {
	mcListProjects()
}

// add new file to existing dataset ID in Material Commons
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

// helper function to publish dataset ID within Material Commons project
func mcPublish(did int64) {
	pid := getMcProjectId()
	_, err := mcClient.PublishDataset(pid, int(did))
	exit("unable to publish dataset", err)
	ds, err := mcClient.MintDOIForDataset(pid, int(did))
	exit("unable to mint DOI for dataset", err)
	fmt.Printf("Dataset has been published...\n")
	fmt.Printf("ID       : %+v\n", ds.ID)
	fmt.Printf("UUID     : %+v\n", ds.UUID)
	fmt.Printf("Name     : %+v\n", ds.Name)
	fmt.Printf("DOI      : %+v\n", ds.DOI)
	fmt.Printf("Created  : %+v\n", ds.CreatedAt)
	fmt.Printf("Published: %+v\n", ds.PublishedAt)
}
