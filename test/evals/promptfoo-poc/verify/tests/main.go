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
	FilePath        string `yaml:"filePath"`
	PreBuildState   string `yaml:"preBuildState"`
	Changes         string `yaml:"changes"`
	PostBuildState  string `yaml:"postBuildState"`
	Diffs           string `yaml:"diffs"`
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

	// Generate the changes file path
	fileNameWithoutExt := strings.TrimSuffix(fileName, fileExt)
	
	// Write the postBuildState content to a file
	postBuildStatePath := filepath.Join(codeDir, fileNameWithoutExt+".post"+fileExt)
	err = os.WriteFile(postBuildStatePath, []byte(entry.Vars.PostBuildState), 0644)
	if err != nil {
		fmt.Printf("Error writing postBuildState file: %v\n", err)
		return
	}
	
	changesFilePath := filepath.Join(changesDir, fileNameWithoutExt+".changes.md")
	changesContent := formatChanges(entry.Vars.Changes)

	// Write the changes content to the changes file
	err = os.WriteFile(changesFilePath, []byte(changesContent), 0644)
	if err != nil {
		fmt.Printf("Error writing changes file: %v\n", err)
		return
	}

	// Generate the diffs file path
	diffsFilePath := filepath.Join(changesDir, fileNameWithoutExt+".diff.txt")

	// Write the diffs content to the diffs file
	err = os.WriteFile(diffsFilePath, []byte(entry.Vars.Diffs), 0644)
	if err != nil {
		fmt.Printf("Error writing diffs file: %v\n", err)
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
