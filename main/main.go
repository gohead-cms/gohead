package main

import (
	"log"

	"gitlab.com/sudo.bngz/gohead/cmd"
)

func main() {
	// Execute the root Cobra command
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
