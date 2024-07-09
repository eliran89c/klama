package cmd

import (
	"os"

	"github.com/eliran89c/klama/internal/app"
	"github.com/eliran89c/klama/internal/app/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "klama [flags] \"prompt\"",
		Short: "Klama is an AI-powered DevOps assistant.",
		Long: `Klama is a CLI tool that helps diagnose and troubleshoot DevOps-related issues 
using AI-powered assistance. It interacts with multiple language models to interpret 
user queries, validate and execute commands, and provide insights 
based on the results.`,
	}

	k8sCmd = &cobra.Command{
		Use:   "k8s",
		Short: "Interact with the Kubernetes debugging assistant",
		Long: `Interact with the Kubernetes debugging assistant to troubleshoot and resolve issues in
Kubernetes clusters.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			debug := viper.GetBool("debug")
			return app.Run(debug, types.AgentTypeKubernetes, types.ExecuterTypeTerminal)
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(k8sCmd)

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
