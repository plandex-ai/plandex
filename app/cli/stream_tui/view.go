package streamtui

import (
	"fmt"
	"sort"
	"strings"

	"plandex/term"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

var borderColor = lipgloss.Color("#444")
var helpTextColor = lipgloss.Color("#ddd")

func (m streamUIModel) View() string {
	if m.promptingMissingFile {
		return m.renderMissingFilePrompt()
	}

	var views []string
	if !m.buildOnly {
		views = append(views, m.renderMainView())
	}
	if m.processing || m.starting {
		views = append(views, m.renderProcessing())
	}
	if m.building {
		views = append(views, m.renderBuild())
	}
	views = append(views, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func (m streamUIModel) renderMainView() string {
	return m.mainViewport.View()
}

func (m streamUIModel) renderHelp() string {
	style := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color(helpTextColor)).BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color(borderColor))

	if m.buildOnly {
		return style.Render(" (s)top • (b)ackground")
	} else {
		return style.Render(" (s)top • (b)ackground • (j/k) scroll • (d/u) page • (g/G) start/end")
	}
}

func (m streamUIModel) renderProcessing() string {
	if m.starting || m.processing {
		return "\n " + m.spinner.View()
	} else {
		return ""
	}
}

func (m streamUIModel) renderBuild() string {
	return m.doRenderBuild(false)
}

func (m streamUIModel) renderStaticBuild() string {
	return m.doRenderBuild(true)
}

func (m streamUIModel) doRenderBuild(outputStatic bool) string {
	if !m.building && !outputStatic {
		return ""
	}

	if outputStatic && len(m.finishedByPath) == 0 && len(m.tokensByPath) == 0 {
		return ""
	}

	var style lipgloss.Style
	if m.buildOnly {
		style = lipgloss.NewStyle().Width(m.width)
	} else {
		style = lipgloss.NewStyle().Width(m.width).BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color(borderColor))
	}

	lbl := "Building plan "
	bgColor := color.BgGreen
	if outputStatic {
		// log.Printf("m.finished: %v, len(m.finishedByPath): %d, len(m.tokensByPath): %d", m.finished, len(m.finishedByPath), len(m.tokensByPath))

		if m.finished || len(m.finishedByPath) == len(m.tokensByPath) {
			lbl = "Built plan "
		} else if m.stopped || m.err != nil || m.apiErr != nil {
			lbl = "Build incomplete "
			bgColor = color.BgRed
		}
	}

	head := color.New(bgColor, color.FgHiWhite, color.Bold).Sprint(" 🏗  ") + color.New(bgColor, color.FgHiWhite).Sprint(lbl)

	filePaths := make([]string, 0, len(m.tokensByPath))
	for filePath := range m.tokensByPath {
		filePaths = append(filePaths, filePath)
	}

	sort.Strings(filePaths)

	var rows [][]string
	lineWidth := 0
	lineNum := 0
	rowIdx := 0

	for _, filePath := range filePaths {
		tokens := m.tokensByPath[filePath]
		finished := m.finishedByPath[filePath]
		block := fmt.Sprintf("📄 %s", filePath)

		if finished {
			block += " ✅"
		} else if tokens > 0 {
			block += fmt.Sprintf(" %d 🪙", tokens)
		}

		maybePrefix := ""
		if rowIdx > 0 {
			maybePrefix = " | "
		}

		maybeBlockWidth := lipgloss.Width(maybePrefix + block)

		if lineWidth+maybeBlockWidth > m.width {
			lineWidth = 0
			lineNum++
			rowIdx = 0
		} else {
			block = maybePrefix + block
		}

		defBlockWidth := lipgloss.Width(block)

		if len(rows) <= lineNum {
			rows = append(rows, []string{})
		}

		row := rows[lineNum]
		row = append(row, block)
		rows[lineNum] = row

		lineWidth += defBlockWidth
		rowIdx++
	}

	resRows := make([]string, len(rows)+1)

	resRows[0] = head
	for i, row := range rows {
		resRows[i+1] = strings.Join(row, "")
	}

	return style.Render(strings.Join(resRows, "\n"))
}

func (m streamUIModel) renderMissingFilePrompt() string {
	style := lipgloss.NewStyle().Padding(1).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(borderColor)).Width(m.width - 2).Height(m.height - 2)

	prompt := "📄 " + color.New(color.Bold, term.ColorHiYellow).Sprint(m.missingFilePath) + " isn't in context."

	prompt += "\n\n"

	desc := "This file exists in your project, but isn't loaded into context. Unless you load it into context or skip generating it, Plandex will fully overwrite the existing file rather than applying updates."

	words := strings.Split(desc, " ")
	for i, word := range words {
		words[i] = color.New(color.FgWhite).Sprint(word)
	}

	prompt += strings.Join(words, " ")

	prompt += "\n\n" + color.New(term.ColorHiMagenta, color.Bold).Sprintln("🧐 What do you want to do?")

	for i, opt := range missingFileSelectOpts {
		if i == m.missingFileSelectedIdx {
			prompt += color.New(term.ColorHiCyan, color.Bold).Sprint(" > " + opt)
		} else {
			prompt += "   " + opt
		}

		if opt == MissingFileLoadLabel {
			prompt += fmt.Sprintf(" | %d 🪙", m.missingFileTokens)
		}

		prompt += "\n"
	}

	return style.Render(prompt)
}
