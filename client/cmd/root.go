package cmd

import (
	"fmt"
	"os"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile string
	verbose int

	rootCmd = &cobra.Command{
		Use:   "client",
		Short: "client command line client",
		Long: `client command line client
	Complete documentation is available at https://www.lepp.cornell.com/CHESComputing/`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

// client configuration
var _srvConfig *srvConfig.SrvConfig

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.srv.yaml)")
	rootCmd.PersistentFlags().IntVar(&verbose, "verbose", 0, "verbosity level)")

	rootCmd.AddCommand(metaCommand())
	rootCmd.AddCommand(dbsCommand())
	rootCmd.AddCommand(authCommand())
	rootCmd.AddCommand(s3Command())
}

func initConfig() {
	config, err := srvConfig.ParseConfig(cfgFile)
	if err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	_srvConfig = &config
}
