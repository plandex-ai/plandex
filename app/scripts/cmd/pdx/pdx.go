package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <directory>")
		return
	}

	dir := os.Args[1]
	err := filepath.Walk(dir, processFile)
	if err != nil {
		fmt.Printf("Error walking the directory: %v\n", err)
	}
}

func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	// Check if the file matches the pattern <name>.post.<ext>
	matched, err := regexp.MatchString(`.*\.post\..*`, info.Name())
	if err != nil {
		return err
	}

	if matched {
		modifyFile(path)
	}

	return nil
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
