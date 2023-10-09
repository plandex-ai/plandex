package shared

import "strings"

type replyInfo struct {
	lines           []string
	lineIndex       int
	maybeFilePath   string
	currentFilePath string
	files           map[string]bool
	countTokens     bool
	numTokensByFile map[string]int
}

func NewReplyInfo(countTokens bool) *replyInfo {
	info := &replyInfo{
		lines: []string{""},
		files: make(map[string]bool),
	}

	if countTokens {
		info.countTokens = true
		info.numTokensByFile = make(map[string]int)
	}

	return info
}

func (r *replyInfo) AddChunk(chunk string) {
	// fmt.Println("Adding chunk:", strconv.Quote(chunk)) // Logging the chunk that's being processed

	hasNewLine := false
	nextChunk := ""

	if chunk == "\n" {
		// fmt.Println("Chunk is \\n, adding new line")
		r.lines = append(r.lines, "")
		hasNewLine = true
		r.lineIndex++
	} else {
		chunkLines := strings.Split(chunk, "\n")

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
					r.AddChunk(nextChunk)
				}()
			}
		}
	}

	if r.lineIndex == 0 || !hasNewLine {
		return
	}

	prevFullLine := r.lines[r.lineIndex-1]
	// fmt.Println("Previous full line:", strconv.Quote(prevFullLine)) // Logging the full line that's being checked

	if r.maybeFilePath != "" {
		// fmt.Println("Maybe file path is:", r.maybeFilePath) // Logging the maybeFilePath
		if strings.HasPrefix(prevFullLine, "```") {
			r.currentFilePath = r.maybeFilePath
			r.maybeFilePath = ""
			// fmt.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path
		} else if prevFullLine != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks
			r.maybeFilePath = ""
		}
		return
	}

	if r.currentFilePath == "" {
		var gotPath string
		if (strings.HasPrefix(prevFullLine, "-") && strings.HasSuffix(prevFullLine, ":")) || strings.HasPrefix(prevFullLine, "-file:") || strings.HasPrefix(prevFullLine, "- file:") || (strings.HasPrefix(prevFullLine, "**") && strings.HasSuffix(prevFullLine, "**")) {
			p := strings.TrimPrefix(prevFullLine, "**")
			p = strings.TrimPrefix(p, "-")
			p = strings.TrimSpace(p)
			p = strings.TrimSuffix(p, "**")
			p = strings.TrimPrefix(p, "file:")
			p = strings.TrimSuffix(p, ":")
			p = strings.TrimSpace(p)
			gotPath = p
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
		if strings.HasPrefix(prevFullLine, "```") {
			r.currentFilePath = ""
			// fmt.Println("Exited file block.")
		} else {
			r.files[r.currentFilePath] = true

			if r.countTokens {
				tokens := int(GetNumTokens(prevFullLine))
				r.numTokensByFile[r.currentFilePath] += tokens
			}
			// r.contentByFile[r.currentFilePath] += prevFullLine + "\n"
			// fmt.Printf("Added %d tokens to %s\n", tokens, r.currentFilePath) // Logging token addition
		}
	}
}

func (r *replyInfo) FinishAndRead() ([]string, map[string]int) {
	r.AddChunk("\n")

	files := make([]string, 0, len(r.files))
	for file := range r.files {
		files = append(files, file)
	}

	return files, r.numTokensByFile
}
