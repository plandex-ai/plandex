package proposal

import (
	"encoding/json"
	"plandex-server/types"
	"strings"
	"sync"
	"time"

	lorem "github.com/drhodes/golorem"
	"github.com/plandex/plandex/shared"
)

func applyLoremStyling(paragraphs []string) []string {
	for p, paragraph := range paragraphs {
		sentences := strings.Split(paragraph, ". ")
		for _, sentence := range sentences {
			words := strings.Split(sentence, " ")
			for i, word := range words {
				if (i+1)%5 == 0 { // Bold every 5th word
					words[i] = "**" + word + "**"
				}
				if (i+1)%8 == 0 { // Italicize every 8th word
					words[i] = "_" + word + "_"
				}
				if (i+1)%7 == 0 { // Color every 7th word
					words[i] = `<span style="color:blue">` + word + `</span>`
				}
			}
		}
		paragraphs[p] = strings.Join(sentences, ". ")
	}
	return paragraphs
}

// Function to stream "lorem ipsum" text sentence by sentence with delay
func streamLoremIpsum(onStream types.OnStreamFunc) {
	paragraphs := []string{lorem.Paragraph(2, 3), lorem.Paragraph(2, 3), lorem.Paragraph(2, 3)}
	paragraphs = applyLoremStyling(paragraphs)

	for _, paragraph := range paragraphs {
		for _, line := range strings.Split(paragraph, "\n") {
			for _, word := range strings.Split(line, " ") {
				onStream(word+" ", nil)
				time.Sleep(50 * time.Millisecond)
			}
			onStream("\n", nil)
			time.Sleep(50 * time.Millisecond)
		}
		onStream("\n\n", nil)
		time.Sleep(50 * time.Millisecond)
	}
	onStream(shared.STREAM_DESCRIPTION_PHASE, nil)

	planDescription := &shared.PlanDescription{
		MadePlan: true,
		Files:    []string{"file1.txt", "file2.txt"},
		// HasExec:  true,
	}
	planDescriptionBytes, _ := json.Marshal(planDescription)
	planDescriptionJson := string(planDescriptionBytes)
	time.Sleep(2000 * time.Millisecond)

	onStream(planDescriptionJson, nil)

	onStream(shared.STREAM_BUILD_PHASE, nil)
	confirmProposal("", onStream)

}

func streamFilesLoremIpsum(onStream types.OnStreamFunc) {
	writeChunk := func(filePath string, isExec bool, content string) {
		chunk := shared.PlanChunk{FilePath: filePath, Content: content /*IsExec: isExec*/}
		chunkJson, _ := json.Marshal(chunk)
		onStream(string(chunkJson), nil)
		time.Sleep(50 * time.Millisecond) // Adding a small delay between files for effect
	}

	// For each file in the proposal, stream some unstyled lorem ipsum content
	var wg sync.WaitGroup
	files := []string{"file1.txt", "file2.txt"}
	for _, filePath := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			text := strings.Join([]string{lorem.Paragraph(2, 3), lorem.Paragraph(2, 3), lorem.Paragraph(2, 3)}, "\n\n")
			streamedFile := shared.StreamedFile{Content: text}
			fileContent, _ := json.Marshal(streamedFile)

			for _, line := range strings.Split(string(fileContent), "\n") {
				for _, word := range strings.Split(line, " ") {
					writeChunk(filePath, false, word+" ")
				}
				writeChunk(filePath, false, "\n")
			}
		}(filePath)
	}

	// For the exec.sh script, generate a script that echoes some lorem ipsum content

	// 	echoContent := `#!/bin/sh
	// echo "` + lorem.Sentence(5, 6) + `"` +
	// 		`
	// 		echo "` + lorem.Sentence(5, 6) + `"` +
	// 		`
	// 		echo "hi"`

	// 	fmt.Println("exec.sh content: " + echoContent)

	// 	streamedFile := shared.StreamedFile{Content: echoContent}
	// 	fileContent, _ := json.Marshal(streamedFile)

	// 	wg.Add(1)

	// 	go func() {
	// 		defer wg.Done()
	// 		for _, line := range strings.Split(string(fileContent), "\n") {
	// 			for _, word := range strings.Split(line, " ") {
	// 				writeChunk("", true, word+" ")
	// 			}
	// 			writeChunk("", true, "\n")
	// 		}
	// 	}()

	wg.Wait()

	onStream(shared.STREAM_FINISHED, nil)
}
