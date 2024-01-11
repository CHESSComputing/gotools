package cmd

// CHESComputing client tool: config module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// helper function to provide usage of config option
func configUsage() {
	fmt.Println("client config <conifg.yaml>")
}

func printConfig(args []string) {
	var fname string
	home, err := os.UserHomeDir()
	if err == nil {
		fname = filepath.Join(home, ".srv.yaml")
	}
	if len(args) == 1 {
		fname = args[0]
	}
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("ERROR: unable to read config file: %s, error %v", fname, err)
		os.Exit(1)
	}
	var config srvConfig.SrvConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("ERROR: unable to unmarshal data: %s, error %v", string(data), err)
		os.Exit(1)
	}
	fmt.Println(config.String())
}

func configCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "client config commamd",
		Long:  "client config command\n" + doc,
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
