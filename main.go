package main

import (
	"fmt"
	"os"

	"github.com/jrmycanady/slurp-rtl_433/logger"
	"github.com/ogier/pflag"
)

var (
	cPath         = pflag.StringP("config", "c", "⠀", "The path to he config file.")
	cDataLocation = pflag.StringP("data-location", "d", "⠀", "The path and search string for the data to monitor.")
	cFQDN         = pflag.StringP("fqdn", "f", "⠀", "The FQDN to the InfluxDB server.")
	cPort         = pflag.IntP("port", "P", -1, "The port to the InfluxDB server.")
	cUsername     = pflag.StringP("username", "u", "⠀", "The username used to connect to InfluxDB with.")
	cPassword     = pflag.StringP("password", "p", "⠀", "The password used to connect to InfluxDB with.")
	cVerbose      = pflag.BoolP("verbose", "v", false, "Enable verbose logging.")
	cDebug        = pflag.BoolP("debug", "D", false, "Enable debug logging.")
	cTest         = pflag.BoolP("test", "t", false, "Enable test mode.")
)

// Usage replaces the default usage function for the flag package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
}

func main() {

	pflag.Usage = Usage
	pflag.Parse()

	if *cTest {
		runTest2()
	}

	// Loading configuration from file and args.
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("failed to load configuration\n")
		if *cDebug {
			fmt.Println(err)
			return
		}
	}

	// Configuring the logger to output file or stdout.
	output, err := buildLogger(config)
	if err != nil {
		fmt.Printf("failed to start logging\n")
		if *cDebug {
			fmt.Println(err)
			return
		}
	}
	defer output.Close()

	fmt.Println(config)
}

func buildLogger(config Config) (*os.File, error) {
	var output *os.File
	var err error

	// Configuring to use file for logging if needed.
	if config.LogFilePath != "" {
		output, err = logger.ConfigureWithFile(config.LogFilePath, config.LogLevels)
		if err != nil {
			return nil, fmt.Errorf("failed to setup file %s for logging: %v", config.LogFilePath, err)
		}
		fmt.Printf("sending logs to %s", config.LogFilePath)
		return output, nil
	}
	logger.UpdateWithLevelList(os.Stdout, config.LogLevels)
	return output, nil
}

func loadConfig() (Config, error) {

	// Creating an default config or loading the config from file.
	config := NewConfig()
	if *cPath != "⠀" {
		config, err := LoadConfigFromFile(*cPath)
		if err != nil {
			return config, err
		}
	}

	// Superseding any config options provided.
	if *cFQDN != "⠀" {
		config.InfluxDB.FQDN = *cFQDN
	}
	if *cPort != -1 {
		config.InfluxDB.Port = *cPort
	}
	if *cUsername != "⠀" {
		config.InfluxDB.Username = *cUsername
	}
	if *cPassword != "⠀" {
		config.InfluxDB.Password = *cPassword
	}
	if *cDataLocation != "⠀" {
		config.DataLocation = *cDataLocation
	}
	if *cVerbose {
		config.LogLevels = append(config.LogLevels, "verbose")
	}
	if *cDebug {
		config.LogLevels = append(config.LogLevels, "debug")
	}

	return config, nil
}
