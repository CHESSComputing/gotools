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
	chessUserConfig := "/nfs/chess/user/chess_chapaas/.foxden.yaml"
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.foxden.yaml)")
	rootCmd.PersistentFlags().IntVar(&verbose, "verbose", 0, "verbosity level)")
	if cfgFile != "" {
		// will use cfgFile provided via --config option
	} else if os.Getenv("FOXDEN_CONFIG") != "" {
		// use config defined in FOXDEN_CONFIG environment
		cfgFile = os.Getenv("FOXDEN_CONFIG")
	} else if _, err := os.Stat(defaultConfig); err == nil {
		// use config located in user's home area
		cfgFile = defaultConfig
	} else if _, err := os.Stat(chessUserConfig); err == nil {
		// use CHESS based user's config
		cfgFile = chessUserConfig
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
	// check that our config file does not exist
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		msg := fmt.Sprintf("FOXDEN config: '%s' does not exist.\n", cfgFile)
		msg += "Please either use --config=<config> option or define FOXDEN_CONFIG environment with your configuration file"
		log.Fatal(msg)
	}
	// parse our config file
	config, err := srvConfig.ParseConfig(cfgFile)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	srvConfig.Config = &config
	if os.Getenv("FOXDEN_VERBOSE") != "" {
		fmt.Println("FOXDEN uses:", cfgFile)
		fmt.Printf("FOXDEN services: %+v\n", srvConfig.Config.Services)
	}
	verbose := strings.ToLower(fmt.Sprintf("%v", os.Getenv("FOXDEN_VERBOSE")))
	if verbose == "1" || verbose == "true" {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	}
}
