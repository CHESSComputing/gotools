package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	authz "github.com/CHESSComputing/golib/authz"
	"github.com/spf13/cobra"
)

// helper function to provide usage of view option
func viewUsage() {
	fmt.Println("client view <DID>")
}

// helper function to print view data records in Json format
func viewMetaRecord(user, did string) {
	query := "did:" + did
	records, err := metaRecords(user, query)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	for _, r := range records {
		fmt.Println("---")
		fmt.Println("### MetaData records:")
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			exit("unable to marshal data", err)
		}
		fmt.Println(string(data))
	}
}

// helper function to look-up DBS records
func viewDBSRecord(user, did string) {
	// look-up dataset records
	rurl := fmt.Sprintf("%s/datasets?did=%s", _srvConfig.Services.DataBookkeepingURL, did)
	resp, err := _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from search-data service", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from search-data service", err)
	}
	fmt.Println("### Provenance dataset records:")
	fmt.Println(string(data))

	// look-up files records
	rurl = fmt.Sprintf("%s/files?did=%s", _srvConfig.Services.DataBookkeepingURL, did)
	resp, err = _httpReadRequest.Get(rurl)
	if err != nil {
		exit("unable to fetch data from search-data service", err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		exit("unable to read data from search-data service", err)
	}
	fmt.Println("### Provenance files records:")
	fmt.Println(string(data))
}

func viewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "client view command",
		Long:  "client view-data command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			token, err := accessToken()
			if err != nil {
				exit("unable to get access token", err)
			}
			claims := authz.TokenClaims(token, _srvConfig.Authz.ClientID)
			rclaims := claims.RegisteredClaims
			user := rclaims.Subject
			if len(args) == 0 {
				viewUsage()
			} else {
				viewMetaRecord(user, args[0])
				viewDBSRecord(user, args[0])
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		viewUsage()
		return nil
	})
	return cmd
}
