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
		fmt.Println("pdx_app <directory>")
		return
	}

	dir := os.Args[1]

	// detect if we are in a directory or a file
	fileInfo, err := os.Stat(dir)
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}

	if fileInfo.IsDir() {
		fmt.Print("Processing directory\n")
		err := filepath.Walk(dir, processFile)
		if err != nil {
			fmt.Printf("Error walking the directory: %v\n", err)
		}
	} else {
		fmt.Print("Processing file\n")
		err = processFile(dir, fileInfo, nil)
		if err != nil {
			fmt.Printf("Error processing file %s: %v\n", dir, err)
		}
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
	}
	modifyFile(path)

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
