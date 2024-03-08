package cmd

// CHESComputing foxden tool: config module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/spf13/cobra"
)

// helper function to provide usage of config option
func configUsage() {
	fmt.Println("foxden config <conifg.yaml>")
}

func printConfig(args []string) {
	var fname, cfile string
	home, err := os.UserHomeDir()
	if err == nil {
		fname = filepath.Join(home, ".foxden.yaml")
	}
	// determine which configuration file we use
	if _, err := os.Stat(fname); err == nil {
		cfile = fname
	} else {
		if _, err := os.Stat(os.Getenv("FOXDEN_CONFIG")); err == nil {
			cfile = os.Getenv("FOXDEN_CONFIG")
		} else {
			fname = "/nfs/chess/user/chess_chapaas/.foxden.yaml"
			if _, err := os.Stat(fname); err == nil {
				cfile = fname
			} else {
				fname = `\\chesssamba.classe.cornell.edu\user\chess_chapaas\.foxden.yaml`
				if _, err := os.Stat(fname); err == nil {
					cfile = fname
				} else {
					msg := "FOXDEN configuration file is not found"
					log.Fatal(msg)
				}
			}
		}
	}
	config, err := srvConfig.ParseConfig(cfile)
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
	fmt.Printf("Configuration file: %s\n", cfile)
	fmt.Println(config.String())
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
