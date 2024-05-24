package cmd

// CHESComputing foxden tool: describe module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"strings"

	"github.com/CHESSComputing/golib/ql"
	schema "github.com/CHESSComputing/golib/schema"
	"github.com/spf13/cobra"
)

var _metaManager *schema.MetaDataManager

// helper function to provide usage of describe option
func describeUsage() {
	fmt.Println("foxden describe <key>")
	fmt.Println("options: --show=<description, service, schema, units, data-type>\n")
	fmt.Println("Examples: \n")
	fmt.Println("# show full details about beam_energy\n")
	fmt.Println("foxden describe beam_energy")
	fmt.Println("# show only units of beam_energy\n")
	fmt.Println("foxden describe beam_energy --show=units")
	fmt.Println("# show which services provides did\n")
	fmt.Println("foxden describe did --show=services")
}

func describeKey(args []string, show string) {
	qlRecords, err := ql.QLRecords("")
	if err != nil {
		exit("unable to get FOXDEN QL keys", err)
	}
	var records []MapRecord
	for _, qrec := range qlRecords {
		for _, key := range args {
			if strings.Contains(qrec.Key, key) {
				rec := make(MapRecord)
				rec[qrec.Key] = qrec.Details(show)
				records = append(records, rec)
			}
		}
	}
	printRecords(records, show)

}

func describeMetaKey(key string) {
	fmt.Println("not implemented yet")
}

func describeProvenanceKey(key string) {
	fmt.Println("not implemented yet")
}

func describeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "foxden describe command",
		Long:  "foxden describe meta-data command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			show, _ := cmd.Flags().GetString("show")
			if len(args) == 0 {
				describeUsage()
			} else {
				describeKey(args, show)
			}
		},
	}
	cmd.PersistentFlags().String("show", "", "show specific part, e.g. description, service, schema, units, data-type")
	cmd.SetUsageFunc(func(*cobra.Command) error {
		describeUsage()
		return nil
	})
	return cmd
}
