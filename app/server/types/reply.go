package types

import (
	"log"
	"strings"
)

type replyParser struct {
	lines            []string
	currentFileLines []string
	lineIndex        int
	maybeFilePath    string
	currentFilePath  string
	currentFileIdx   int
	files            []string
	fileContents     []string
	numTokens        int
	numTokensByFile  map[string]int
}

func NewReplyParser() *replyParser {
	info := &replyParser{
		lines:            []string{""},
		currentFileLines: []string{},
		files:            []string{},
		fileContents:     []string{},
		numTokensByFile:  make(map[string]int),
	}

	return info
}

func (r *replyParser) AddChunk(chunk string, addToTotal bool) {
	// log.Println("Adding chunk:", strconv.Quote(chunk)) // Logging the chunk that's being processed

	hasNewLine := false
	nextChunk := ""

	if addToTotal {
		r.numTokens++

		if r.currentFilePath != "" {
			r.numTokensByFile[r.currentFilePath]++
		}

		// log.Println("Total tokens:", r.numTokens)
		// log.Println("Tokens by file path:", r.numTokensByFile)
	}

	if chunk == "\n" {
		// log.Println("Chunk is \\n, adding new line")
		r.lines = append(r.lines, "")
		hasNewLine = true
		r.lineIndex++
	} else {
		chunkLines := strings.Split(chunk, "\n")

		// log.Println("Chunk lines:", len(chunkLines))

		currentLine := r.lines[r.lineIndex]
		currentLine += chunkLines[0]

		// log.Println("Current line:", strconv.Quote(currentLine))
		r.lines[r.lineIndex] = currentLine

		if len(chunkLines) > 1 {
			r.lines = append(r.lines, chunkLines[1])
			r.lineIndex++
			hasNewLine = true

			if len(chunkLines) > 2 {
				tail := chunkLines[2:]
				nextChunk = "\n" + strings.Join(tail, "\n")
				defer func() {
					// log.Println("Recursive add next queued chunk:", strconv.Quote(nextChunk))
					r.AddChunk(nextChunk, false)
				}()
			}
		}
	}

	if r.lineIndex == 0 || !hasNewLine {
		// log.Println("No new line detected--returning")
		return
	}

	prevFullLine := r.lines[r.lineIndex-1]
	// log.Println("Previous full line:", strconv.Quote(prevFullLine)) // Logging the full line that's being checked

	prevFullLineTrimmed := strings.TrimSpace(prevFullLine)

	if r.maybeFilePath != "" {
		// log.Println("Maybe file path is:", r.maybeFilePath) // Logging the maybeFilePath
		if strings.HasPrefix(prevFullLineTrimmed, "```") {
			r.currentFilePath = r.maybeFilePath
			r.currentFileIdx = len(r.files)
			r.fileContents = append(r.fileContents, "")
			r.maybeFilePath = ""
			r.currentFileLines = []string{}
			log.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path

			return
		} else if prevFullLineTrimmed != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks
			r.maybeFilePath = ""
		}
	}

	if r.currentFilePath == "" {
		// log.Println("Current file path is empty--checking for possible file path...")

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
			// log.Println("No possible file path detected.", strconv.Quote(prevFullLineTrimmed))
		}

		if gotPath != "" {
			// log.Println("Detected possible file path:", gotPath) // Logging the possible file path

			if r.maybeFilePath == "" {
				r.maybeFilePath = gotPath
			} else {
				r.maybeFilePath = gotPath
			}
		}
	} else {
		log.Println("Current file path is not empty--adding to current file...")

		if strings.HasPrefix(prevFullLineTrimmed, "```") {
			log.Println("Found closing ticks--adding file to files and resetting current file...")
			r.files = append(r.files, r.currentFilePath)
			r.currentFilePath = ""
		} else {
			log.Println("Adding tokens to current file...") // Logging token addition

			r.fileContents[r.currentFileIdx] += prevFullLine + "\n"
			r.currentFileLines = append(r.currentFileLines, prevFullLine)
			// fmt.Printf("Added %d tokens to %s\n", tokens, r.currentFilePath) // Logging token addition
		}
	}
}

func (r *replyParser) Read() (files []string, fileContents []string, numTokensByFile map[string]int, totalTokens int) {
	return r.files, r.fileContents, r.numTokensByFile, r.numTokens
}

func (r *replyParser) FinishAndRead() (files []string, fileContents []string, numTokensByFile map[string]int, totalTokens int) {
	r.AddChunk("\n", false)
	return r.Read()
}
