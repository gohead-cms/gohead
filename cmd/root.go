package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gohead",
	Short: "GoHead is a headless CMS and agentic framework.",
	Long: `A flexible and powerful headless CMS built with Go,
featuring an agentic framework for building autonomous workflows.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}
