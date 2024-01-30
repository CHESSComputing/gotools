package cmd

// CHESComputing client tool: root module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"log"
	"os"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile string
	verbose int

	rootCmd = &cobra.Command{
		Use:   "client",
		Short: "client command line tool",
		Long:  "client command line tool\n" + doc,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

// client configuration
var _srvConfig *srvConfig.SrvConfig

func init() {
	_httpReadRequest = services.NewHttpRequest("read", 0)
	_httpWriteRequest = services.NewHttpRequest("write", 0)
	_httpDeleteRequest = services.NewHttpRequest("delete", 0)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.foxden.yaml)")
	rootCmd.PersistentFlags().IntVar(&verbose, "verbose", 0, "verbosity level)")
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(metaCommand())
	rootCmd.AddCommand(searchCommand())
	rootCmd.AddCommand(dbsCommand())
	rootCmd.AddCommand(authCommand())
	rootCmd.AddCommand(s3Command())
	rootCmd.AddCommand(mlCommand())
	rootCmd.AddCommand(viewCommand())
	rootCmd.AddCommand(syncCommand())
	rootCmd.AddCommand(configCommand())
}

func initConfig() {
	config, err := srvConfig.ParseConfig(cfgFile)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	_srvConfig = &config
	log.SetFlags(log.LstdFlags | log.Llongfile)
}
