package cmd

import (
	"errors"
	"fmt"

	utils "github.com/CHESSComputing/golib/utils"
	"github.com/spf13/cobra"
)

// helper function to determine which DOI provider to use
func doiProvider() string {
	var provider string
	return provider
}

// helper function to view document ID in DOI provider
func doiView(did int64) {
	provider := doiProvider()
	if provider == "Zenodo" {
		zenodoView(did)
	} else if provider == "MaterialCommons" {
		mcView(did)
	} else {
		exit("Unsupported DOI provider", errors.New("unsupported"))
	}
}

// helper function to publish document ID in DOI provider
func doiPublish(did int64) {
	provider := doiProvider()
	if provider == "Zenodo" {
		zenodoView(did)
	} else if provider == "MaterialCommons" {
		mcView(did)
	} else {
		exit("Unsupported DOI provider", errors.New("unsupported"))
	}
}

// helper function to update document ID with file in DOI provider
func doiUpdate(did int64, fname string) {
	provider := doiProvider()
	if provider == "Zenodo" {
		zenodoUpdate(did, fname)
	} else if provider == "MaterialCommons" {
		mcUpdate(did, fname)
	} else {
		exit("Unsupported DOI provider", errors.New("unsupported"))
	}
}

// helper function to add document ID and file in DOI provider
func doiAdd(did int64, fname string) {
	provider := doiProvider()
	if provider == "Zenodo" {
		zenodoAdd(did, fname)
	} else if provider == "MaterialCommons" {
		mcAdd(did, fname)
	} else {
		exit("Unsupported DOI provider", errors.New("unsupported"))
	}
}

// helper function to create document in DOI provider via provided file
func doiCreate(fname string) {
	provider := doiProvider()
	if provider == "Zenodo" {
		zenodoCreate(fname)
	} else if provider == "MaterialCommons" {
		mcCreate(fname)
	} else {
		exit("Unsupported DOI provider", errors.New("unsupported"))
	}
}

// helper function to list documents in DOI provider (optionally view specific document id)
func doiDocs(did int64) {
	provider := doiProvider()
	if provider == "Zenodo" {
		zenodoDocs(did)
	} else if provider == "MaterialCommons" {
		mcDocs(did)
	} else {
		exit("Unsupported DOI provider", errors.New("unsupported"))
	}
}

// helper function to provide doi usage info
func doiUsage() {
	fmt.Println("foxden doi <ls|create|update|publish|view> <DID> [options]")
	fmt.Println("options: file name")
	fmt.Println("         --ztoken=<zenodo token or file>")
	fmt.Println("\nExamples:")
	fmt.Println("\n# create new document (new document with some ID, e.g. 123456789, will be created)")
	fmt.Println("foxden doi create")
	fmt.Println("\n# the out of above command will be like")
	fmt.Println("      Document is created: id=123456789 URL=https://zenodo.org/deposit/123456789")
	fmt.Println("\n# create new document with user's zenodo token")
	fmt.Println("foxden doi create --ztoken=alksjdfkds")
	fmt.Println("\n# create new document from given record:")
	fmt.Println("foxden doi create </path/record.json>")
	fmt.Println("\n# add file to document id:")
	fmt.Println("foxden doi add <id> </path/regular/file>")
	fmt.Println("\n# update document id with publish data record:")
	fmt.Println("foxden doi update <id> /path/record.json")
	fmt.Println("\n# publish document id:")
	fmt.Println("foxden doi publish <id>")
	fmt.Println("\n# list existing documents:")
	fmt.Println("foxden doi ls <id>")
	fmt.Println("\n# get details of document id:")
	fmt.Println("foxden doi view <id>")
	fmt.Println()
	fmt.Println("Here is example of record.json")
	record := `
{
    "files": [
        {"name": "file1.txt", "file": "/path/file1.txt"},
        {"name": "file2.txt", "file": "/path/file2.txt"}
    ],
    "metadata": {
        "publication_type": "article",
        "upload_type": "publication",
        "description": "Test FOXDEN publication",
        "keywords": ["bla", "foo"],
        "creators": [{"name": "First Last", "affiliation": "Zenodo"}],
        "title": "Test experiment"
    }
}`
	fmt.Println(record)
}

func printDoiRecord(rec map[string]any) {
	maxLen := 20
	if val, ok := rec["id"]; ok {
		key := utils.PaddedKey("id", maxLen)
		vvv := val.(float64)
		v := int64(vvv)
		fmt.Printf("%s: %v\n", key, v)
	}
	if val, ok := rec["links"]; ok {
		vvv := val.(map[string]any)
		if v, ok := vvv["html"]; ok {
			key := utils.PaddedKey("URL", maxLen)
			fmt.Printf("%s: %v\n", key, v)
		}
	}
}

func doiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doi",
		Short: "foxden doi command",
		Long:  "foxden doi command to access FOXDEN Publication service\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			tkn, _ := cmd.Flags().GetString("ztoken")
			initZenodoAccess(tkn)
			if len(args) == 0 {
				doiUsage()
			} else if args[0] == "ls" {
				accessToken()
				var did int64
				if len(args) == 2 {
					did = getDID(args[1])
				}
				doiDocs(did)
			} else if args[0] == "create" {
				accessToken()
				writeToken()
				var fname string
				if len(args) == 2 {
					fname = args[1]
				}
				doiCreate(fname)
			} else if args[0] == "add" {
				accessToken()
				writeToken()
				did, fname := getParams(args[1:])
				doiAdd(did, fname)
			} else if args[0] == "update" {
				accessToken()
				writeToken()
				did, fname := getParams(args[1:])
				doiUpdate(did, fname)
			} else if args[0] == "publish" {
				accessToken()
				writeToken()
				did := getDID(args[1])
				doiPublish(did)
			} else if args[0] == "view" {
				accessToken()
				did := getDID(args[1])
				doiView(did)
			} else {
				fmt.Printf("WARNING: unsupported option(s) %+v\n", args)
			}
		},
	}
	cmd.PersistentFlags().String("ztoken", "", "zenodo token file or token string")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		doiUsage()
		return nil
	})
	return cmd
}
