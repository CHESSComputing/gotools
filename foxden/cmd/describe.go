package cmd

// CHESComputing foxden tool: describe module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"

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
	schemas := []string{"ID1A3", "ID3A", "ID4B"}
	for _, sname := range schemas {
		umap := _metaManager.Units(sname)
		dmap := _metaManager.Descriptions(sname)
		fmt.Printf("Schema: %s\n", sname)
		for _, key := range args {
			if val, ok := dmap[key]; ok {
				fmt.Printf("%s: %s\n", key, val)
			}
			if val, ok := umap[key]; ok {
				if val != "" {
					fmt.Printf("%s: units in %s\n", key, val)
				}
			}
		}
	}
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
