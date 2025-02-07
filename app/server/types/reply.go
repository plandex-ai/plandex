package types

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	shared "plandex-shared"
)

const verboseLogging = false

type ReplyParserRes struct {
	MaybeFilePath   string
	CurrentFilePath string
	IsInMoveBlock   bool
	IsInRemoveBlock bool
	IsInResetBlock  bool
	Operations      []*shared.Operation
	TotalTokens     int
}

type ReplyParser struct {
	lines                     []string
	currentFileLines          []string
	lineIndex                 int
	maybeFilePath             string
	currentFilePath           string
	currentDescriptionLines   []string
	currentDescriptionLineIdx int
	numTokens                 int
	operations                []*shared.Operation
	currentFileOperation      *shared.Operation
	pendingOperations         []*shared.Operation
	pendingPaths              map[string]bool
	isInMoveBlock             bool
	isInRemoveBlock           bool
	isInResetBlock            bool
}

func NewReplyParser() *ReplyParser {
	info := &ReplyParser{
		lines:                   []string{""},
		currentFileLines:        []string{},
		currentDescriptionLines: []string{""},
		operations:              []*shared.Operation{},
		pendingPaths:            map[string]bool{},
	}
	return info
}

func (r *ReplyParser) AddChunk(chunk string, addToTotal bool) {
	if verboseLogging {
		log.Println("Adding chunk:", strconv.Quote(chunk)) // Logging the chunk that's being processed
	}

	hasNewLine := false
	nextChunk := ""

	if addToTotal {
		r.numTokens++
		// log.Println("Total tokens:", r.numTokens)
		// log.Println("Tokens by file path:", r.numTokensByFile)
	}

	if r.currentFilePath != "" && r.currentFileOperation != nil {
		r.currentFileOperation.NumTokens++
	}

	if chunk == "\n" {
		if verboseLogging {
			log.Println("Chunk is \\n, adding new line")
		}
		r.lines = append(r.lines, "")
		hasNewLine = true
		r.lineIndex++

		if r.currentFileOperation == nil {
			if verboseLogging {
				log.Println("Current file operation is empty--adding new description line...")
			}
			r.currentDescriptionLines = append(r.currentDescriptionLines, "")
			r.currentDescriptionLineIdx++
		}

	} else {
		chunkLines := strings.Split(chunk, "\n")

		if verboseLogging {
			log.Println("Chunk lines:", len(chunkLines))
		}

		currentLine := r.lines[r.lineIndex]
		currentLine += chunkLines[0]
		if verboseLogging {
			log.Println("Current line:", strconv.Quote(currentLine))
		}
		r.lines[r.lineIndex] = currentLine

		if r.currentFileOperation == nil {
			if verboseLogging {
				log.Println("Current file operation is empty--adding to current description...")
				log.Println("Current description lines:", r.currentDescriptionLines)
				log.Printf("Current description line index: %d\n", r.currentDescriptionLineIdx)
			}

			currentDescLine := r.currentDescriptionLines[r.currentDescriptionLineIdx]
			currentDescLine += chunkLines[0]
			r.currentDescriptionLines[r.currentDescriptionLineIdx] = currentDescLine
		}

		if len(chunkLines) > 1 {
			r.lines = append(r.lines, chunkLines[1])
			r.lineIndex++

			if r.currentFileOperation == nil {
				if verboseLogging {
					log.Println("Current file operation is empty--adding to current description...")
					log.Println("Current description lines:", r.currentDescriptionLines)
					log.Printf("Current description line index: %d\n", r.currentDescriptionLineIdx)
				}

				r.currentDescriptionLines = append(r.currentDescriptionLines, chunkLines[1])
				r.currentDescriptionLineIdx++
			}

			hasNewLine = true

			if len(chunkLines) > 2 {
				tail := chunkLines[2:]
				nextChunk = "\n" + strings.Join(tail, "\n")
				defer func() {
					if verboseLogging {
						log.Println("Recursive add next queued chunk:", strconv.Quote(nextChunk))
					}
					r.AddChunk(nextChunk, false)
				}()
			}
		}
	}

	if r.lineIndex == 0 || !hasNewLine {
		if verboseLogging {
			log.Println("No new line detected--returning")
		}
		return
	}

	prevFullLine := r.lines[r.lineIndex-1]
	if verboseLogging {
		log.Println("Previous full line:", strconv.Quote(prevFullLine)) // Logging the full line that's being checked
	}

	prevFullLineTrimmed := strings.TrimSpace(prevFullLine)

	setCurrentFile := func(path string, noLabel bool) {
		r.currentFilePath = path
		r.currentFileOperation = &shared.Operation{
			Type: shared.OperationTypeFile,
			Path: path,
		}
		r.maybeFilePath = ""
		r.currentFileLines = []string{}

		var fileDescription string
		skipNumLines := 4
		if noLabel {
			skipNumLines = 2
		}
		if len(r.currentDescriptionLines) > skipNumLines {
			fileDescription = strings.TrimSpace(strings.Join(r.currentDescriptionLines[0:len(r.currentDescriptionLines)-skipNumLines], "\n"))
			if verboseLogging {
				log.Println("File description:", fileDescription)
			}
			if fileDescription != "" {
				r.currentFileOperation.Description = fileDescription
			}
		} else {
			r.currentFileOperation.Description = ""
		}

		r.currentDescriptionLines = []string{""}
		r.currentDescriptionLineIdx = 0

		if verboseLogging {
			log.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path
		}

	}

	if r.maybeFilePath != "" && !r.isInMoveBlock && !r.isInRemoveBlock && !r.isInResetBlock {
		if verboseLogging {
			log.Println("Maybe file path is:", r.maybeFilePath) // Logging the maybeFilePath
		}
		if strings.HasPrefix(prevFullLineTrimmed, "<PlandexBlock") {
			if verboseLogging {
				log.Println("Found opening tag--confirming file path...") // Logging the confirmed file path
			}

			setCurrentFile(r.maybeFilePath, false)
			return
		} else if prevFullLineTrimmed != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks

			if verboseLogging {
				log.Println("Previous maybeFilePath was not a file path--resetting maybeFilePath")
			}

			r.maybeFilePath = ""
		}
	}

	if r.currentFilePath == "" && !r.isInMoveBlock && !r.isInRemoveBlock && !r.isInResetBlock {
		if verboseLogging {
			log.Println("Current file path is empty--checking for possible file path...")
		}

		if LineHasXmlPath(prevFullLineTrimmed) {
			if verboseLogging {
				log.Println("Line has XML-style PlandexBlock tag")
			}
			path := extractFilePath(prevFullLineTrimmed)
			if path != "" {
				setCurrentFile(path, true)
			}
		}

		var gotPath string
		if LineMaybeHasFilePath(prevFullLineTrimmed) {
			gotPath = extractFilePath(prevFullLineTrimmed)
		} else if prevFullLineTrimmed == "### Move Files" {
			if verboseLogging {
				log.Println("Found move block")
			}
			r.isInMoveBlock = true
		} else if prevFullLineTrimmed == "### Remove Files" {
			if verboseLogging {
				log.Println("Found remove block")
			}
			r.isInRemoveBlock = true
		} else if prevFullLineTrimmed == "### Reset Changes" {
			if verboseLogging {
				log.Println("Found reset block")
			}
			r.isInResetBlock = true
		}

		if gotPath != "" {
			if verboseLogging {
				log.Println("Detected possible file path:", gotPath) // Logging the possible file path
			}
			r.maybeFilePath = gotPath
		}
	} else if r.currentFilePath != "" {
		if verboseLogging {
			log.Println("Current file path is not empty--adding to current file...")
		}
		if prevFullLineTrimmed == "</PlandexBlock>" {
			if verboseLogging {
				log.Println("Found closing tag--adding file to files and resetting current file...")
			}
			r.operations = append(r.operations, r.currentFileOperation)
			r.currentFilePath = ""
			r.currentFileOperation = nil

		} else {
			if verboseLogging {
				log.Println("Adding tokens to current file...") // Logging token addition
			}

			r.currentFileOperation.Content += prevFullLine + "\n"
			r.currentFileLines = append(r.currentFileLines, prevFullLine)

		}
	} else if r.isInMoveBlock || r.isInRemoveBlock || r.isInResetBlock {
		if verboseLogging {
			log.Println("In move, remove, or reset block")
		}
		if prevFullLineTrimmed == "<EndPlandexFileOps/>" {
			if verboseLogging {
				log.Println("Found closing tag--adding operations to operations and resetting pending operations...")
			}
			r.isInMoveBlock = false
			r.isInRemoveBlock = false
			r.isInResetBlock = false
			r.operations = append(r.operations, r.pendingOperations...)
			r.pendingOperations = []*shared.Operation{}
			r.pendingPaths = map[string]bool{}
		} else if r.isInMoveBlock {
			op := extractMoveFile(prevFullLineTrimmed)
			if op != nil && !r.pendingPaths[op.Path] {
				if verboseLogging {
					log.Println("Found move operation")
				}
				r.pendingOperations = append(r.pendingOperations, op)
				r.pendingPaths[op.Path] = true
			}
		} else if r.isInRemoveBlock {
			op := extractRemoveOrResetFile(shared.OperationTypeRemove, prevFullLineTrimmed)
			if op != nil && !r.pendingPaths[op.Path] {
				if verboseLogging {
					log.Println("Found remove operation")
				}
				r.pendingOperations = append(r.pendingOperations, op)
				r.pendingPaths[op.Path] = true
			}
		} else if r.isInResetBlock {
			op := extractRemoveOrResetFile(shared.OperationTypeReset, prevFullLineTrimmed)
			if op != nil && !r.pendingPaths[op.Path] {
				if verboseLogging {
					log.Println("Found reset operation")
				}
				r.pendingOperations = append(r.pendingOperations, op)
				r.pendingPaths[op.Path] = true
			}
		}
	}
}

func (r *ReplyParser) Read() ReplyParserRes {
	return ReplyParserRes{
		MaybeFilePath:   r.maybeFilePath,
		CurrentFilePath: r.currentFilePath,
		Operations:      r.operations,
		IsInMoveBlock:   r.isInMoveBlock,
		IsInRemoveBlock: r.isInRemoveBlock,
		IsInResetBlock:  r.isInResetBlock,
		TotalTokens:     r.numTokens,
	}
}

func (r *ReplyParser) FinishAndRead() ReplyParserRes {
	r.AddChunk("\n", false)
	return r.Read()
}

func (r *ReplyParser) GetReplyBeforeCurrentPath() string {
	return r.GetReplyBeforePath(r.currentFilePath)
}

func (r *ReplyParser) GetReplyBeforePath(path string) string {
	if path == "" {
		return strings.Join(r.lines, "\n")
	}

	var idx int
	for i := len(r.lines) - 1; i >= 0; i-- {
		line := r.lines[i]
		if LineMaybeHasFilePath(line) && path == extractFilePath(line) {
			idx = i
			break
		}
	}

	return strings.Join(r.lines[:idx], "\n")
}

func (r *ReplyParser) GetReplyForMissingFile() string {
	path := r.currentFilePath

	var idx int
	for i := len(r.lines) - 1; i >= 0; i-- {
		line := r.lines[i]
		if LineMaybeHasFilePath(line) && path == extractFilePath(line) {
			idx = i
			break
		}
	}

	if idx == -1 {
		return strings.Join(r.lines, "\n")
	}

	idx = idx + 2

	if idx > len(r.lines)-1 {
		return strings.Join(r.lines, "\n")
	}

	return strings.Join(r.lines[:idx], "\n") + "\n"
}

func (r *ReplyParserRes) FileOperationBlockOpen() bool {
	return r.IsInMoveBlock || r.IsInRemoveBlock || r.IsInResetBlock
}

func LineHasXmlPath(line string) bool {
	return strings.HasPrefix(line, "<PlandexBlock") && strings.Contains(line, `path="`)
}

func LineMaybeHasFilePath(line string) bool {
	couldBe := (strings.HasPrefix(line, "-")) || strings.HasPrefix(line, "-file:") || strings.HasPrefix(line, "- file:") || (strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**")) || (strings.HasPrefix(line, "#") && strings.HasSuffix(line, ":"))

	if couldBe {
		extracted := extractFilePath(line)

		extSplit := strings.Split(extracted, ".")
		hasExt := len(extSplit) > 1 && !strings.Contains(extSplit[len(extSplit)-1], " ")
		hasFileSep := strings.Contains(extracted, string(os.PathSeparator))
		hasSpaces := strings.Contains(extracted, " ")

		return !(!hasExt && !hasFileSep && hasSpaces)
	}

	return couldBe
}

var re = regexp.MustCompile(`path="([^"]+)"`)

func extractFilePath(line string) string {
	// Handle XML-style PlandexBlock tag
	if strings.HasPrefix(line, "<PlandexBlock") {
		match := re.FindStringSubmatch(line)
		if len(match) > 1 {
			return match[1]
		}
		return ""
	}

	p := strings.ReplaceAll(line, "**", "")
	p = strings.ReplaceAll(p, "`", "")
	p = strings.ReplaceAll(p, "'", "")
	p = strings.ReplaceAll(p, `"`, "")
	p = strings.TrimPrefix(p, "-")
	p = strings.TrimPrefix(p, "####")
	p = strings.TrimPrefix(p, "###")
	p = strings.TrimPrefix(p, "##")
	p = strings.TrimPrefix(p, "#")
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "file:")
	p = strings.TrimPrefix(p, "file path:")
	p = strings.TrimPrefix(p, "filepath:")
	p = strings.TrimPrefix(p, "File path:")
	p = strings.TrimPrefix(p, "File Path:")
	p = strings.TrimSuffix(p, ":")
	p = strings.TrimSpace(p)

	// split := strings.Split(p, " ")
	// if len(split) > 1 {
	// 	p = split[0]
	// }

	split := strings.Split(p, ": ")
	if len(split) > 1 {
		p = split[len(split)-1]
	}

	split = strings.Split(p, " (")
	if len(split) > 1 {
		p = split[0]
	}

	return p
}

func extractMoveFile(line string) *shared.Operation {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "-") {
		return nil
	}

	// Remove the leading dash and trim
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)

	parts := strings.Split(line, "→")
	if len(parts) != 2 {
		return nil
	}

	src := strings.TrimSpace(parts[0])
	dst := strings.TrimSpace(parts[1])

	// Remove backticks
	src = strings.Trim(src, "`")
	dst = strings.Trim(dst, "`")

	return &shared.Operation{
		Type:        shared.OperationTypeMove,
		Path:        src,
		Destination: dst,
	}
}

func extractRemoveOrResetFile(opType shared.OperationType, line string) *shared.Operation {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "-") {
		return nil
	}

	// Remove the leading dash and trim
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)

	path := strings.Trim(line, "`")

	return &shared.Operation{
		Type: opType,
		Path: path,
	}
}
