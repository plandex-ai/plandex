package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Entry struct {
	Vars Vars `yaml:"vars"`
}

type Vars struct {
	FilePath       string `yaml:"filePath"`
	PreBuildInState string `yaml:"preBuildInState"`
	Changes        string `yaml:"changes"`
}

func main() {
	// Read the YAML file
	yamlFile, err := os.ReadFile("languages.test.yml")
	if err != nil {
		fmt.Printf("Error reading YAML file: %v\n", err)
		return
	}

	// Unmarshal the YAML data
	var entries []Entry
	err = yaml.Unmarshal(yamlFile, &entries)
	if err != nil {
		fmt.Printf("Error unmarshalling YAML data: %v\n", err)
		return
	}

	// Process each entry
	for _, entry := range entries {
		_, fileName := filepath.Split(entry.Vars.FilePath)
		fileExt := filepath.Ext(fileName)
		language := strings.TrimPrefix(fileExt, ".")

		// Create the directory paths
		codeDir := filepath.Join("..", "assets", language, "code")
		changesDir := filepath.Join("..", "assets", language, "changes")

		// Ensure the directories exist
		os.MkdirAll(codeDir, os.ModePerm)
		os.MkdirAll(changesDir, os.ModePerm)

		// Generate the source file path
		sourceFilePath := filepath.Join(codeDir, fileName)

		// Write the formatted preBuildInState content to the source file
		formattedPreBuildContent := formatCode(entry.Vars.PreBuildInState)

		err = os.WriteFile(sourceFilePath, []byte(formattedPreBuildContent), 0644)
		if err != nil {
			fmt.Printf("Error writing source file: %v\n", err)
			return
		}

		// Write the preBuildInState content to the source file
		/* preBuildContent := strings.ReplaceAll(entry.Vars.PreBuildInState, "\n", "")
		preBuildContent = strings.ReplaceAll(preBuildContent, "\t", "")

		err = os.WriteFile(sourceFilePath, []byte(preBuildContent), 0644)
		if err != nil {
			fmt.Printf("Error writing source file: %v\n", err)
			return
		} */

		// Format the Go source file if applicable
		//if fileExt == ".go" {
		//	cmd := exec.Command("gofmt", "-w", sourceFilePath)
		//	err = cmd.Run()
		//	if err != nil {
		//		fmt.Printf("Error formatting Go source file: %v\n", err)
		//		return
		//	}
		//}

		// Generate the changes file path
		fileNameWithoutExt := strings.TrimSuffix(fileName, fileExt)
		changesFilePath := filepath.Join(changesDir, fileNameWithoutExt+".changes.md")

		changesContent := formatChanges(entry.Vars.Changes)

		// Write the changes content to the changes file
		err = os.WriteFile(changesFilePath, []byte(changesContent), 0644)
		if err != nil {
			fmt.Printf("Error writing changes file: %v\n", err)
			return
		}
	}

	fmt.Println("Files have been successfully generated.")
}

// formatCode formats the code content with pdx-<line num>: prefix
func formatCode(code string) string {
	var formattedCode strings.Builder
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		formattedCode.WriteString(fmt.Sprintf("pdx-%d: %s\n", i+1, line))
	}

	return formattedCode.String()
}

// formatChanges formats the changes content into the specified markdown format
func formatChanges(changes string) string {
	var formattedChanges strings.Builder
	lines := strings.Split(changes, "\n")
	subtaskCounter := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			if strings.HasPrefix(line, strconv.Itoa(subtaskCounter) + ".") {
				formattedChanges.WriteString(fmt.Sprintf("### Subtask %d: %s\n\n", subtaskCounter, strings.TrimPrefix(line, strconv.Itoa(subtaskCounter) + ".")))
				subtaskCounter++
			} else {
				formattedChanges.WriteString(fmt.Sprintf("%s\n\n", line))
			}
		}
	}

	return formattedChanges.String()
}
