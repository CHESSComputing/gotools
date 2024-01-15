package cmd

// CHESComputing client tool: sync module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"

	"github.com/spf13/cobra"
)

// helper function to provide usage of sync option
func syncUsage() {
	fmt.Println("client sync <service: meta or dbs> URL1 URL2")
}

func syncMetaRecords(url1, url2 string) {
	fmt.Println("not implemented yet")
}

func syncDBSRecords(url1, url2 string) {
	fmt.Println("not implemented yet")
}

func syncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "client sync command",
		Long:  "client sync-data command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			writeToken()
			if len(args) == 0 {
				syncUsage()
			} else if args[1] == "meta" {
				syncMetaRecords(args[2], args[3])
			} else if args[1] == "dbs" {
				syncDBSRecords(args[2], args[3])
			} else {
				syncUsage()
			}
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		syncUsage()
		return nil
	})
	return cmd
}
