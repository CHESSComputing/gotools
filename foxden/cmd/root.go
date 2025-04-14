package cmd

// CHESComputing foxden tool: root module
//
// Copyright (c) 2023 - Valentin Kuznetsov <vkuznet@gmail.com>
//
import (
	"fmt"
	"log"
	"os"
	"strings"

	srvConfig "github.com/CHESSComputing/golib/config"
	services "github.com/CHESSComputing/golib/services"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile string
	verbose int

	rootCmd = &cobra.Command{
		Use:   "foxden",
		Short: "foxden command line tool",
		Long:  "foxden command line tool\n" + doc,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	defaultConfig := fmt.Sprintf("%s/.foxden.yaml", os.Getenv("HOME"))
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.foxden.yaml)")
	rootCmd.PersistentFlags().IntVar(&verbose, "verbose", 0, "verbosity level)")
	if cfgFile != "" {
		// will use cfgFile
	} else if os.Getenv("FOXDEN_CONFIG") != "" {
		cfgFile = os.Getenv("FOXDEN_CONFIG")
	} else {
		cfgFile = defaultConfig
	}
	os.Setenv("FOXDEN_CONFIG", cfgFile)
	cobra.OnInitialize(initConfig)

	_httpReadRequest = services.NewHttpRequest("read", 0)
	_httpWriteRequest = services.NewHttpRequest("write", 0)
	_httpDeleteRequest = services.NewHttpRequest("delete", 0)

	rootCmd.AddCommand(s3Command())
	rootCmd.AddCommand(dmCommand())
	rootCmd.AddCommand(mlCommand())
	rootCmd.AddCommand(doiCommand())
	rootCmd.AddCommand(metaCommand())
	rootCmd.AddCommand(provCommand())
	rootCmd.AddCommand(specCommand())
	rootCmd.AddCommand(authCommand())
	rootCmd.AddCommand(viewCommand())
	rootCmd.AddCommand(syncCommand())
	rootCmd.AddCommand(searchCommand())
	rootCmd.AddCommand(globusCommand())
	rootCmd.AddCommand(configCommand())
	rootCmd.AddCommand(versionCommand())
	rootCmd.AddCommand(describeCommand())
}

func initConfig() {
	if os.Getenv("FOXDEN_VERBOSE") != "" {
		fmt.Println("FOXDEN uses:", cfgFile)
	}
	config, err := srvConfig.ParseConfig(cfgFile)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	srvConfig.Config = &config
	verbose := strings.ToLower(fmt.Sprintf("%v", os.Getenv("FOXDEN_VERBOSE")))
	if verbose == "1" || verbose == "true" {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	}
}
