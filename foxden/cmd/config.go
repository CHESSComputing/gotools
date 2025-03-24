package cmd

// CHESComputing foxden tool: config module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/spf13/cobra"
)

// helper function to provide usage of config option
func configUsage() {
	fmt.Println("foxden config <conifg.yaml>")
}

func printConfig(args []string) {
	fmt.Printf("Configuration file: %s\n", cfgFile)
	fmt.Println(srvConfig.Config.String())
}

func configCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "foxden config commamd",
		Long:  "foxden config command\n" + doc,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			printConfig(args)
		},
	}
	cmd.SetUsageFunc(func(*cobra.Command) error {
		configUsage()
		return nil
	})
	return cmd
}
