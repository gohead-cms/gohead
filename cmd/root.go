package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gohead",
	Short: "Gohead - headless CMS",
	Long:  `Gohead is a headless CMS built to provide flexible content management.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}
