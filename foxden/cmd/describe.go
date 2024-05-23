package cmd

// CHESComputing foxden tool: describe module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"strings"

	ql "github.com/CHESSComputing/golib/ql"
	schema "github.com/CHESSComputing/golib/schema"
	"github.com/spf13/cobra"
)

var _metaManager *schema.MetaDataManager

// helper function to provide usage of describe option
func describeUsage() {
	fmt.Println("foxden describe <key>")
	fmt.Println("Examples: \n")
	fmt.Println("foxden describe beam_energy")
}

func describeKey(args []string) {
	qlKeys, err := ql.QLKeys("")
	if err != nil {
		exit("unable to get FOXDEN QL keys", err)
	}
	var records []MapRecord
	for _, elem := range qlKeys {
		for _, key := range args {
			if strings.Contains(elem, key) {
				arr := strings.Split(elem, ":")
				rec := make(MapRecord)
				rec[arr[0]] = strings.Join(arr[1:len(arr)], ":")
				records = append(records, rec)
			}
		}
	}
	printRecords(records)

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
			if len(args) == 0 {
				describeUsage()
			} else {
				describeKey(args)
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		describeUsage()
		return nil
	})
	return cmd
}
