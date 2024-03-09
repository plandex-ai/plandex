package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

// Version will be set at build time using -ldflags
var Version = "development"
func readVersionFromFile() string {
	file, err := os.Open("version.txt") // Adjust the path as necessary
	if err != nil {
		fmt.Println("Unable to read version from file, defaulting to:", Version)
		return Version
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}

	return Version
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Plandex",
	Long:  `All software has versions. This is Plandex's`,
	Run: func(cmd *cobra.Command, args []string) {
		if Version == "development" {
			Version = readVersionFromFile()
		}
		fmt.Println("Plandex CLI Version:", Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
