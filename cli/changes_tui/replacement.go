package changes_tui

import (
	"strings"

	"github.com/fatih/color"
	"github.com/muesli/reflow/wrap"
)

const prependLines = 10
const appendLines = 10

func (m changesUIModel) getReplacementOldDisplay() (string, string, string) {
	oldContent := m.selectionInfo.currentRep.Old
	originalFile := m.selectionInfo.currentPlanBeforeReplacement.CurrentPlanFiles.Files[m.selectionInfo.currentPath]

	// log.Printf("oldContent: %v", oldContent)
	// log.Printf("originalFile: %v", originalFile)

	fileIdx := strings.Index(originalFile, oldContent)
	if fileIdx == -1 {
		panic("old content not found in full file") // should never happen
	}

	toPrepend := ""
	numLinesPrepended := 0
	for i := fileIdx - 1; i >= 0; i-- {
		s := string(originalFile[i])
		toPrepend = s + toPrepend
		if originalFile[i] == '\n' {
			numLinesPrepended++
			if numLinesPrepended == prependLines {
				break
			}
		}
	}

	toPrepend = strings.TrimLeft(toPrepend, "\n")

	toAppend := ""
	numLinesAppended := 0
	for i := fileIdx + len(oldContent); i < len(originalFile); i++ {
		s := string(originalFile[i])
		if s == "\t" {
			s = "  "
		}
		toAppend += s
		if originalFile[i] == '\n' {
			numLinesAppended++
			if numLinesAppended == appendLines {
				break
			}
		}
	}

	toAppend = strings.TrimRight(toAppend, "\n")

	wrapWidth := m.changeOldViewport.Width - 6
	toPrepend = wrap.String(toPrepend, wrapWidth)
	oldContent = wrap.String(oldContent, wrapWidth)
	toAppend = wrap.String(toAppend, wrapWidth)

	toPrependLines := strings.Split(toPrepend, "\n")
	for i, line := range toPrependLines {
		toPrependLines[i] = color.New(color.FgWhite).Sprint(line)
	}
	toPrepend = strings.Join(toPrependLines, "\n")

	oldContentLines := strings.Split(oldContent, "\n")
	for i, line := range oldContentLines {
		oldContentLines[i] = color.New(color.FgHiRed).Sprint(line)
	}
	oldContent = strings.Join(oldContentLines, "\n")

	toAppendLines := strings.Split(toAppend, "\n")
	for i, line := range toAppendLines {
		toAppendLines[i] = color.New(color.FgWhite).Sprint(line)
	}
	toAppend = strings.Join(toAppendLines, "\n")

	oldDisplayContent :=
		toPrepend +
			oldContent +
			toAppend

	return oldDisplayContent, toPrepend, toAppend
}

func (m changesUIModel) getReplacementNewDisplay(prependContent, appendContent string) string {
	newContent := m.selectionInfo.currentRep.New
	newContent = wrap.String(newContent, m.changeNewViewport.Width-6)

	newContentLines := strings.Split(newContent, "\n")
	for i, line := range newContentLines {
		newContentLines[i] = color.New(color.FgHiGreen).Sprint(line)
	}
	newContent = strings.Join(newContentLines, "\n")

	return prependContent + newContent + appendContent
}
