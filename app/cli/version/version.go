package version

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Version will be set at build time using -ldflags
var Version = "source"
var BuiltFromSource = false

func init() {
	if Version == "source" {
		BuiltFromSource = true
		readVersionFromFile()
	}
}

func readVersionFromFile() {
	file, err := os.Open("../version.txt")
	if err != nil {
		fmt.Println("Unable to read version from file, defaulting to:", Version)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		Version = strings.TrimSpace(scanner.Text())
	}
}
