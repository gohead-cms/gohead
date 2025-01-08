package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version   string
	commit    string
	buildDate string
)

// Initialize version command
func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Gohead",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Gohead version: %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Build Date: %s\n", buildDate)
	},
}
