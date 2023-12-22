package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plandex/cmd"
)

func init() {
	// set up a file logger
	// TODO: log rotation
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	dir := filepath.Join(home, ".plandex-home")
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err := os.OpenFile(filepath.Join(dir, "plandex.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		os.Exit(1)
	}

	// Set the output of the logger to the file
	log.SetOutput(file)
}

func main() {
	cmd.Execute()
}
