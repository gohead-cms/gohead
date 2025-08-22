package main

import (
	"log"

	"github.com/gohead-cms/gohead/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
