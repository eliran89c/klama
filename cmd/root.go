package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "klama [flags] \"prompt\"",
	Short: "Klama is an AI-powered Kubernetes debugging assistant",
	Long: `Klama is a CLI tool that helps diagnose and troubleshoot Kubernetes-related issues 
using AI-powered assistance. It interacts with multiple language models to interpret 
user queries, validate and execute safe Kubernetes commands, and provide insights 
based on the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			// Show usage if no prompt is provided
			cmd.Help()
			return
		}
		prompt := args[0]
		debug := viper.GetBool("debug")
		showUsage := viper.GetBool("show-usage")

		run(prompt, debug, showUsage)
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.klama.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().Bool("show-usage", false, "Show usage information")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("show-usage", rootCmd.PersistentFlags().Lookup("show-usage"))
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
