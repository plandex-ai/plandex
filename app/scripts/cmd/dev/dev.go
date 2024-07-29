package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run script.go <input_directory> <output_directory>")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	// Check if the input directory exists
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		fmt.Printf("Error: input directory %s does not exist\n", inputDir)
		os.Exit(1)
	}

	// Check if the output directory exists, create if it doesn't
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Error creating output directory %s: %v\n", outputDir, err)
			os.Exit(1)
		}
	}

	// Walk through the files in the input directory
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for the corresponding ".post" file
		if !strings.Contains(info.Name(), ".post.") {
			baseName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			ext := filepath.Ext(info.Name())
			postFile := filepath.Join(inputDir, baseName+".post"+ext)

			if _, err := os.Stat(postFile); err == nil {
				// Prepare the output file name
				outputFileName := fmt.Sprintf("%s.diff.txt", baseName)
				outputFilePath := filepath.Join(outputDir, outputFileName)

				// Prepare the diff command
				cmd := exec.Command("diff", "-u", path, postFile)

				// Redirect the output to the output file
				outfile, err := os.Create(outputFilePath)
				if err != nil {
					fmt.Printf("Error creating output file %s: %v\n", outputFilePath, err)
					return err
				}
				defer outfile.Close()

				cmd.Stdout = outfile
				cmd.Stderr = os.Stderr

				// Run the command
				err = cmd.Run()
				if err != nil {
					if exitError, ok := err.(*exec.ExitError); ok {
						// Check the exit code
						if exitError.ExitCode() == 1 {
							// diff found differences
							fmt.Printf("Differences found and written to %s\n", outputFilePath)
						} else {
							fmt.Printf("Error running diff command on %s and %s: %v\n", path, postFile, err)
							return err
						}
					} else {
						fmt.Printf("Error running diff command on %s and %s: %v\n", path, postFile, err)
						return err
					}
				} else {
					fmt.Printf("No differences found between %s and %s\n", path, postFile)
				}

				// Now modify the original file
				err = modifyFile(path)
				if err != nil {
					fmt.Printf("Error modifying file %s: %v\n", path, err)
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %s: %v\n", inputDir, err)
		os.Exit(1)
	}
}

func modifyFile(filePath string) error {
	inputFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFilePath := filePath + ".tmp"
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	writer := bufio.NewWriter(outputFile)

	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		newLine := fmt.Sprintf("pdx-%d: %s\n", lineNum, line)
		if _, err := writer.WriteString(newLine); err != nil {
			return err
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	if err := inputFile.Close(); err != nil {
		return err
	}

	if err := outputFile.Close(); err != nil {
		return err
	}

	// Replace the original file with the modified file
	if err := os.Rename(outputFilePath, filePath); err != nil {
		return err
	}

	return nil
}
