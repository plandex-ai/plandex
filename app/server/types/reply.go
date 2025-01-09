package types

import (
	"log"
	"os"
	"strings"

	"github.com/plandex/plandex/shared"
)

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
	// log.Println("Adding chunk:", strconv.Quote(chunk)) // Logging the chunk that's being processed

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
		// log.Println("Chunk is \\n, adding new line")
		r.lines = append(r.lines, "")
		hasNewLine = true
		r.lineIndex++

		if r.currentFilePath == "" {
			r.currentDescriptionLines = append(r.currentDescriptionLines, "")
			r.currentDescriptionLineIdx++
		}

	} else {
		chunkLines := strings.Split(chunk, "\n")

		// log.Println("Chunk lines:", len(chunkLines))

		currentLine := r.lines[r.lineIndex]
		currentLine += chunkLines[0]
		// log.Println("Current line:", strconv.Quote(currentLine))
		r.lines[r.lineIndex] = currentLine

		if r.currentFileOperation == nil {
			// log.Println("Current file path is empty--adding to current description...")
			// log.Println("Current description lines:", r.currentDescriptionLines)
			// log.Printf("Current description line index: %d\n", r.currentDescriptionLineIdx)

			currentDescLine := r.currentDescriptionLines[r.currentDescriptionLineIdx]
			currentDescLine += chunkLines[0]
			r.currentDescriptionLines[r.currentDescriptionLineIdx] = currentDescLine
		}

		if len(chunkLines) > 1 {
			r.lines = append(r.lines, chunkLines[1])
			r.lineIndex++

			if r.currentFilePath == "" {
				r.currentDescriptionLines = append(r.currentDescriptionLines, chunkLines[1])
				r.currentDescriptionLineIdx++
			}

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

	if r.maybeFilePath != "" && !r.isInMoveBlock && !r.isInRemoveBlock && !r.isInResetBlock {
		// log.Println("Maybe file path is:", r.maybeFilePath) // Logging the maybeFilePath
		if strings.HasPrefix(prevFullLineTrimmed, "<PlandexBlock") {
			// log.Println("Found opening tag--confirming file path...") // Logging the confirmed file path

			r.currentFilePath = r.maybeFilePath
			r.currentFileOperation = &shared.Operation{
				Type: shared.OperationTypeFile,
				Path: r.maybeFilePath,
			}
			r.maybeFilePath = ""
			r.currentFileLines = []string{}

			var fileDescription string
			if len(r.currentDescriptionLines) > 4 {
				fileDescription = strings.Join(r.currentDescriptionLines[0:len(r.currentDescriptionLines)-4], "\n")
				r.currentFileOperation.Description = fileDescription
			} else {
				r.currentFileOperation.Description = ""
			}

			r.currentDescriptionLines = []string{""}
			r.currentDescriptionLineIdx = 0

			// log.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path

			return
		} else if prevFullLineTrimmed != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks
			r.maybeFilePath = ""
		}
	}

	if r.currentFilePath == "" && !r.isInMoveBlock && !r.isInRemoveBlock && !r.isInResetBlock {
		// log.Println("Current file path is empty--checking for possible file path...")

		var gotPath string
		if LineMaybeHasFilePath(prevFullLineTrimmed) {
			gotPath = extractFilePath(prevFullLineTrimmed)
		} else if prevFullLineTrimmed == "### Move Files" {
			log.Println("Found move block")
			r.isInMoveBlock = true
		} else if prevFullLineTrimmed == "### Remove Files" {
			log.Println("Found remove block")
			r.isInRemoveBlock = true
		} else if prevFullLineTrimmed == "### Reset Changes" {
			log.Println("Found reset block")
			r.isInResetBlock = true
		}

		if gotPath != "" {
			log.Println("Detected possible file path:", gotPath) // Logging the possible file path
			r.maybeFilePath = gotPath
		}
	} else if r.currentFilePath != "" {
		// log.Println("Current file path is not empty--adding to current file...")
		if prevFullLineTrimmed == "</PlandexBlock>" {
			// log.Println("Found closing tag--adding file to files and resetting current file...")
			r.operations = append(r.operations, r.currentFileOperation)
			r.currentFilePath = ""

			// spew.Dump(r.files)
		} else {
			// log.Println("Adding tokens to current file...") // Logging token addition

			r.currentFileOperation.Content += prevFullLine + "\n"
			r.currentFileLines = append(r.currentFileLines, prevFullLine)
			// log.Printf("Added %d tokens to %s\n", tokens, r.currentFilePath) // Logging token addition
		}
	} else if r.isInMoveBlock || r.isInRemoveBlock || r.isInResetBlock {
		log.Println("In move, remove, or reset block")
		if prevFullLineTrimmed == "<EndPlandexFileOps/>" {
			// log.Println("Found closing tag--adding operations to operations and resetting pending operations...")
			r.isInMoveBlock = false
			r.isInRemoveBlock = false
			r.isInResetBlock = false
			r.operations = append(r.operations, r.pendingOperations...)
			r.pendingOperations = []*shared.Operation{}
			r.pendingPaths = map[string]bool{}
		} else if r.isInMoveBlock {
			op := extractMoveFile(prevFullLineTrimmed)
			if op != nil && !r.pendingPaths[op.Path] {
				// log.Println("Found move operation")
				r.pendingOperations = append(r.pendingOperations, op)
				r.pendingPaths[op.Path] = true
			}
		} else if r.isInRemoveBlock {
			op := extractRemoveOrResetFile(shared.OperationTypeRemove, prevFullLineTrimmed)
			if op != nil && !r.pendingPaths[op.Path] {
				// log.Println("Found remove operation")
				r.pendingOperations = append(r.pendingOperations, op)
				r.pendingPaths[op.Path] = true
			}
		} else if r.isInResetBlock {
			op := extractRemoveOrResetFile(shared.OperationTypeReset, prevFullLineTrimmed)
			if op != nil && !r.pendingPaths[op.Path] {
				// log.Println("Found reset operation")
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

func extractFilePath(line string) string {
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
