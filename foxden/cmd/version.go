package cmd

// CHESComputing foxden tool: version module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

func version() {
	goVersion := runtime.Version()
	tstamp := time.Now()
	fmt.Printf("git={{VERSION}} go=%s date=%s\n", goVersion, tstamp)
}

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "foxden version commamd",
		Long:  "foxden version command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			version()
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Println("foxden version")
		return nil
	})
	return cmd
}
