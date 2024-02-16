package types

import (
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type parserRes struct {
	CurrentFilePath string
	Files           []string
	FileContents    []string
	NumTokensByFile map[string]int
	TotalTokens     int
}

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
			// log.Println("Found opening ticks--confirming file path...") // Logging the confirmed file path

			r.currentFilePath = r.maybeFilePath
			r.currentFileIdx = len(r.files)
			r.fileContents = append(r.fileContents, "")
			r.maybeFilePath = ""
			r.currentFileLines = []string{}
			// log.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path

			return
		} else if prevFullLineTrimmed != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks
			r.maybeFilePath = ""
		}
	}

	if r.currentFilePath == "" {
		// log.Println("Current file path is empty--checking for possible file path...")

		var gotPath string
		if lineHasFilePath(prevFullLineTrimmed) {
			gotPath = extractFilePath(prevFullLineTrimmed)
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
		// log.Println("Current file path is not empty--adding to current file...")
		if strings.HasPrefix(prevFullLineTrimmed, "```") {
			// log.Println("Found closing ticks--adding file to files and resetting current file...")
			r.files = append(r.files, r.currentFilePath)
			r.currentFilePath = ""

			spew.Dump(r.files)
		} else {
			// log.Println("Adding tokens to current file...") // Logging token addition

			r.fileContents[r.currentFileIdx] += prevFullLine + "\n"
			r.currentFileLines = append(r.currentFileLines, prevFullLine)
			// log.Printf("Added %d tokens to %s\n", tokens, r.currentFilePath) // Logging token addition
		}
	}
}

func (r *replyParser) Read() parserRes {
	return parserRes{
		CurrentFilePath: r.currentFilePath,
		Files:           r.files,
		FileContents:    r.fileContents,
		NumTokensByFile: r.numTokensByFile,
		TotalTokens:     r.numTokens,
	}
}

func (r *replyParser) FinishAndRead() parserRes {
	r.AddChunk("\n", false)
	return r.Read()
}

func (r *replyParser) GetReplyBeforeCurrentPath() string {
	if r.currentFilePath == "" {
		return strings.Join(r.lines, "\n")
	}

	var idx int
	for i := len(r.lines) - 1; i >= 0; i-- {
		line := r.lines[i]
		if lineHasFilePath(line) && r.currentFilePath == extractFilePath(line) {
			idx = i
			break
		}
	}

	return strings.Join(r.lines[:idx], "\n")
}

func lineHasFilePath(line string) bool {
	return (strings.HasPrefix(line, "-")) || strings.HasPrefix(line, "-file:") || strings.HasPrefix(line, "- file:") || (strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**"))
}

func extractFilePath(line string) string {
	p := strings.ReplaceAll(line, "**", "")
	p = strings.TrimPrefix(p, "-")
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "file:")
	p = strings.TrimSuffix(p, ":")
	p = strings.TrimSpace(p)

	split := strings.Split(p, " ")
	if len(split) > 1 {
		p = split[0]
	}

	return p
}
