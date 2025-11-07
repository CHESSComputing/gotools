package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/CHESSComputing/golib/globus"
)

func initConfig(cfgFile string) {
	chessUserConfig := "/nfs/chess/user/chess_chapaas/.foxden.yaml"
	if cfgFile != "" {
		// will use cfgFile provided via --config option
	} else if os.Getenv("FOXDEN_CONFIG") != "" {
		// use config defined in FOXDEN_CONFIG environment
		cfgFile = os.Getenv("FOXDEN_CONFIG")
	} else if _, err := os.Stat(chessUserConfig); err == nil {
		// use CHESS based user's config
		cfgFile = chessUserConfig
	}
	os.Setenv("FOXDEN_CONFIG", cfgFile)
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
}

func main() {
	var path string
	flag.StringVar(&path, "path", "", "data location path")
	var globusDir string
	flag.StringVar(&globusDir, "globusDir", "CHESS Raw", "globus area")
	var cfgFile string
	flag.StringVar(&cfgFile, "cfgFile", "", "FOXDEN configuration file")
	flag.Parse()
	initConfig(cfgFile)
	gurl, err := globus.ChessGlobusLink(globusDir, path)
	if err == nil {
		fmt.Println(gurl)
	} else {
		fmt.Println("ERROR:", err)
	}
}
