package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path/to/directory>", os.Args[0])
	}

	dirPath := os.Args[1]
	dirName := filepath.Base(dirPath)

	// Create the main directory
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Fatalf("Error creating directory: %s", err)
	}

	f, err := os.Create(fmt.Sprintf("%s/%s", dirPath, "promptfooconfig.yaml"))
	if err != nil {
		log.Fatalf("Error creating file: %s", err)
	}
	f.Close()

	// Create files inside the directory
	files := []string{
		"parameters.json",
		"config.properties",
		"prompt.txt",
	}

	for _, file := range files {
		f, err := os.Create(fmt.Sprintf("%s/%s.%s", dirPath, dirName, file))
		if err != nil {
			log.Fatalf("Error creating file: %s", err)
		}
		f.Close()
	}

	// Create assets and tests directories
	subDirs := []string{"assets", "tests"}

	for _, subDir := range subDirs {
		if err := os.Mkdir(fmt.Sprintf("%s/%s", dirPath, subDir), 0755); err != nil {
			log.Fatalf("Error creating subdirectory: %s", err)
		}
	}

	// Template for promptfooconfig.yaml
	ymlTemplate := `description: "{{ .Name }}"

prompts:
  - file://{{ .Name }}.prompt.txt

providers:
  - file://{{ .Name }}.provider.yml

tests: tests/*.tests.yml
`

	// Populate promptfooconfig.yaml
	promptFooConfigTmpl, err := template.New("yml").Parse(ymlTemplate)
	if err != nil {
		log.Fatalf("Error creating template: %s", err)
	}

	// Template for config.properties
	propertiesTemplate := `provider_id=openai:gpt-4o
temperature=
max_tokens=
top_p=
response_format=
function_name=
tool_type=function
function_param_type=object
tool_choice_type=function
tool_choice_function_name=
nested_parameters_json={{ .Name }}.parameters.json
`

	// Populate config.properties
	configPropertiesTmpl, err := template.New("properties").Parse(propertiesTemplate)
	if err != nil {
		log.Fatalf("Error creating template: %s", err)
	}

	configFile, err := os.Create(fmt.Sprintf("%s/%s.%s", dirPath, dirName, "config.properties"))
	if err != nil {
		log.Fatalf("Error creating config.properties: %s", err)
	}
	defer configFile.Close()

	file, err := os.Create(fmt.Sprintf("%s/promptfooconfig.yaml", dirPath))
	if err != nil {
		log.Fatalf("Error creating promptfooconfig.yaml: %s", err)
	}
	defer file.Close()

	data := struct {
		Name string
	}{
		Name: dirName,
	}

	if err := promptFooConfigTmpl.Execute(file, data); err != nil {
		log.Fatalf("Error executing template: %s", err)
	}

	if err := configPropertiesTmpl.Execute(configFile, data); err != nil {
		log.Fatalf("Error executing template: %s", err)
	}

	fmt.Println("Directory created successfully!")
	fmt.Println("Please check the contents of the directory and proceed with the implementation.")
}
