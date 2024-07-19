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
	FilePath               string `yaml:"filePath"`
	PreBuildState          string `yaml:"preBuildState"`
	Changes                string `yaml:"changes"`
	IncorrectlyUpdatedFile string `yaml:"incorrectlyUpdatedFile"`
	Problems               string `yaml:"problems"`
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
		processEntry(entry)
	}

	fmt.Println("Files have been successfully generated.")
}

func processEntry(entry Entry) {
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

	// Write the formatted preBuildState content to the source file
	formattedPreBuildContent := formatCode(entry.Vars.PreBuildState)
	err := os.WriteFile(sourceFilePath, []byte(formattedPreBuildContent), 0644)
	if err != nil {
		fmt.Printf("Error writing source file: %v\n", err)
		return
	}

	// Write the incorrectlyUpdatedFile content to a file
	incorrectlyUpdatedFilePath := filepath.Join(codeDir, "incorrectly_updated_file"+fileExt)
	err = os.WriteFile(incorrectlyUpdatedFilePath, []byte(entry.Vars.IncorrectlyUpdatedFile), 0644)
	if err != nil {
		fmt.Printf("Error writing incorrectlyUpdatedFile file: %v\n", err)
		return
	}

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

	// Generate the problems file path
	problemsFilePath := filepath.Join(changesDir, fileNameWithoutExt+".problems.md")

	// Write the problems content to the problems file
	err = os.WriteFile(problemsFilePath, []byte(entry.Vars.Problems), 0644)
	if err != nil {
		fmt.Printf("Error writing problems file: %v\n", err)
		return
	}
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

func formatChanges(changes string) string {
	var formattedChanges strings.Builder
	lines := strings.Split(changes, "\n")
	subtaskCounter := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, strconv.Itoa(subtaskCounter)+".") {
			formattedChanges.WriteString(fmt.Sprintf("### Subtask %d: %s\n\n", subtaskCounter, strings.TrimPrefix(line, strconv.Itoa(subtaskCounter)+".")))
			subtaskCounter++
		} else {
			formattedChanges.WriteString(fmt.Sprintf("%s\n\n", line))
		}
	}

	return formattedChanges.String()
}
