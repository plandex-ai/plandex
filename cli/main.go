package main

import (
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

	dir := filepath.Join(home, ".plandex")
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	file, err := os.OpenFile(filepath.Join(dir, "plandex.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// Set the output of the logger to the file
	log.SetOutput(file)
}

func main() {
	cmd.Execute()
}
