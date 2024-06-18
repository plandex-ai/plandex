package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var testDir = "../../test/evals/promptfoo-poc"
var templFile = testDir + "/templates/" + "/provider.template.yml"

func main() {

	testAbsPath, _ := filepath.Abs(testDir)
	templAbsPath, _ := filepath.Abs(templFile)

	// Function to walk through directories and find required values
	err := filepath.Walk(testAbsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".properties" {
			dirName := filepath.Base(filepath.Dir(path))
			outputFileName := filepath.Join(filepath.Dir(path), dirName+".provider.yml")

			// Read the template file
			templateContent, err := os.ReadFile(templAbsPath)
			if err != nil {
				log.Fatalf("Error reading template file: %v", err)
			}

			// Prepare variables (this example assumes properties file is a simple key=value format)
			variables := map[string]interface{}{}
			properties, err := os.ReadFile(path)
			if err != nil {
				log.Fatalf("Error reading properties file: %v", err)
			}
			for _, line := range strings.Split(string(properties), "\n") {
				if len(line) > 0 {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						if key == "nested_parameters_json" {
							// Read the file path from the nested_properties_json key
							propertiesJsonFile := filepath.Join(filepath.Dir(path), value)
							jsonProperties, err := os.ReadFile(propertiesJsonFile)
							if err != nil {
								log.Fatalf("Error reading nested parameters JSON file: %v", err)
							}
							// Parse the JSON string
							var nestedProperties map[string]interface{}

							err = json.Unmarshal(jsonProperties, &nestedProperties)

							if err != nil {
								log.Fatalf("Error unmarshalling nested properties JSON: %v", err)
							}

							parameters, err := json.Marshal(nestedProperties)
							if err != nil {
								log.Fatalf("Error marshalling nested properties JSON: %v", err)
							}

							// Add the nested properties to the variables
							variables["parameters"] = string(parameters)
						} else {
							variables[key] = value
						}
					}
				}
			}

			// Parse and execute the template
			tmpl, err := template.New("yamlTemplate").Parse(string(templateContent))
			if err != nil {
				log.Fatalf("Error parsing template: %v", err)
			}
			outputFile, err := os.Create(outputFileName)
			if err != nil {
				log.Fatalf("Error creating output file: %v", err)
			}
			defer outputFile.Close()

			err = tmpl.Execute(outputFile, variables)
			if err != nil {
				log.Fatalf("Error executing template: %v", err)
			}
			log.Printf("Template rendered and saved to '%s'", outputFileName)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking the path: %v", err)
	}
}
