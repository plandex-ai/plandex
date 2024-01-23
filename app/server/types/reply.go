package types

import (
	"strings"
)

type replyParser struct {
	lines            []string
	currentFileLines []string
	lineIndex        int
	maybeFilePath    string
	currentFilePath  string
	files            map[string]bool
	fileContents     map[string]string
	numTokens        int
	numTokensByFile  map[string]int
}

func NewReplyParser() *replyParser {
	info := &replyParser{
		lines:            []string{""},
		currentFileLines: []string{},
		files:            make(map[string]bool),
		fileContents:     make(map[string]string),
		numTokensByFile:  make(map[string]int),
	}

	return info
}

func (r *replyParser) AddToken(token string, addToTotal bool) {
	// fmt.Println("Adding chunk:", strconv.Quote(chunk)) // Logging the chunk that's being processed

	hasNewLine := false
	nextChunk := ""

	if addToTotal {
		r.numTokens++

		if r.currentFilePath != "" {
			r.numTokensByFile[r.currentFilePath]++
		}
	}

	if token == "\n" {
		// fmt.Println("Chunk is \\n, adding new line")
		r.lines = append(r.lines, "")
		hasNewLine = true
		r.lineIndex++
	} else {
		chunkLines := strings.Split(token, "\n")

		// fmt.Println("Chunk lines:", len(chunkLines))

		currentLine := r.lines[r.lineIndex]
		currentLine += chunkLines[0]

		// fmt.Println("Current line:", strconv.Quote(currentLine))
		r.lines[r.lineIndex] = currentLine

		if len(chunkLines) > 1 {
			r.lines = append(r.lines, chunkLines[1])
			r.lineIndex++
			hasNewLine = true

			if len(chunkLines) > 2 {
				tail := chunkLines[2:]
				nextChunk = "\n" + strings.Join(tail, "\n")
				defer func() {
					// fmt.Println("Recursive add next queued chunk:", strconv.Quote(nextChunk))
					r.AddToken(nextChunk, false)
				}()
			}
		}
	}

	if r.lineIndex == 0 || !hasNewLine {
		return
	}

	prevFullLine := r.lines[r.lineIndex-1]
	// fmt.Println("Previous full line:", strconv.Quote(prevFullLine)) // Logging the full line that's being checked

	prevFullLineTrimmed := strings.TrimSpace(prevFullLine)

	if r.maybeFilePath != "" {
		// fmt.Println("Maybe file path is:", r.maybeFilePath) // Logging the maybeFilePath
		if strings.HasPrefix(prevFullLineTrimmed, "```") {
			r.currentFilePath = r.maybeFilePath
			r.maybeFilePath = ""
			r.currentFileLines = []string{}
			// fmt.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path
			return
		} else if prevFullLineTrimmed != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks
			r.maybeFilePath = ""
		}
	}

	if r.currentFilePath == "" {
		// fmt.Println("Current file path is empty--checking for possible file path...")

		var gotPath string
		if (strings.HasPrefix(prevFullLineTrimmed, "-")) || strings.HasPrefix(prevFullLineTrimmed, "-file:") || strings.HasPrefix(prevFullLineTrimmed, "- file:") || (strings.HasPrefix(prevFullLineTrimmed, "**") && strings.HasSuffix(prevFullLineTrimmed, "**")) {
			p := strings.TrimPrefix(prevFullLineTrimmed, "**")
			p = strings.TrimPrefix(p, "-")
			p = strings.TrimSpace(p)
			p = strings.TrimSuffix(p, "**")
			p = strings.TrimPrefix(p, "file:")
			p = strings.TrimSuffix(p, ":")
			p = strings.TrimSpace(p)
			gotPath = p
		} else {
			// fmt.Println("No possible file path detected.", strconv.Quote(prevFullLineTrimmed))
		}

		if gotPath != "" {
			// fmt.Println("Detected possible file path:", gotPath) // Logging the possible file path

			if r.maybeFilePath == "" {
				r.maybeFilePath = gotPath
			} else {
				r.maybeFilePath = gotPath
			}
		}
	} else {
		if strings.HasPrefix(prevFullLineTrimmed, "```") {
			r.currentFilePath = ""
			// fmt.Println("Exited file block.")
		} else {
			r.files[r.currentFilePath] = true
			r.fileContents[r.currentFilePath] += prevFullLine + "\n"
			r.currentFileLines = append(r.currentFileLines, prevFullLine)

			// r.contentByFile[r.currentFilePath] += prevFullLine + "\n"
			// fmt.Printf("Added %d tokens to %s\n", tokens, r.currentFilePath) // Logging token addition
		}
	}
}

func (r *replyParser) Read() (files []string, fileContents map[string]string, numTokensByFile map[string]int, totalTokens int) {
	r.AddToken("\n", false)

	files = make([]string, 0, len(r.files))
	for file := range r.files {
		files = append(files, file)
	}

	return files, r.fileContents, r.numTokensByFile, r.numTokens
}
