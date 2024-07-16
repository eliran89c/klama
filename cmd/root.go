package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// version information
var (
	version = "dev"
	arch    = "dev"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Short: "Klama is an AI-powered DevOps assistant.",
		Long: `Klama is a CLI tool that helps diagnose and troubleshoot DevOps-related issues 
using AI-powered assistance. It interacts with multiple language models to interpret 
user queries, validate and execute commands, and provide insights 
based on the results.`,
	}
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Klama version %v %v\n", version, arch)
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add subcommands
	rootCmd.AddCommand(k8sCmd)
	rootCmd.AddCommand(versionCmd)

	// add global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.klama.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search in the home directory
		viper.AddConfigPath(home)

		// Also search in the current directory
		viper.AddConfigPath(".")

		viper.SetConfigType("yaml")
		viper.SetConfigName(".klama")
	}

	// Bind environment variables
	viper.AutomaticEnv()

	// Read in config file
	viper.ReadInConfig()
}
