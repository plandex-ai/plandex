package streamtui

import (
	"fmt"
	"sort"
	"strings"

	"plandex-cli/term"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

var borderColor = lipgloss.Color("#444")
var helpTextColor = lipgloss.Color("#ddd")

func (m streamUIModel) View() string {

	if m.promptingMissingFile {
		return m.renderMissingFilePrompt()
	}

	views := []string{}
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
		s := " (s)top"
		if m.canSendToBg {
			s += " • (b)ackground"
		}
		return style.Render(s)
	} else {
		s := " (s)top"
		if m.canSendToBg {
			s += " • (b)ackground"
		}
		s += " • (j/k) scroll • (d/u) page • (g/G) start/end"
		return style.Render(s)
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

	if !outputStatic && m.buildViewCollapsed {
		// Render collapsed view
		inProgress := 0
		total := len(m.tokensByPath)
		for path := range m.tokensByPath {
			if path == "_apply.sh" {
				total--
				continue
			}
			if !m.finishedByPath[path] {
				inProgress++
			}
		}

		_, hasApplyScript := m.tokensByPath["_apply.sh"]
		applyScriptFinished := m.finishedByPath["_apply.sh"]

		lbl := "file"
		if total > 1 {
			lbl = "files"
		}

		var summary string
		if total > 0 {
			summary = fmt.Sprintf(" 📄 %d %s", total, lbl)
		}
		if inProgress > 0 {
			summary += fmt.Sprintf(" • 📝 editing %d %s", inProgress, m.buildSpinner.View())
		}
		if hasApplyScript {
			if total > 0 {
				summary += " •"
			}
			if applyScriptFinished {
				summary += " 🚀 wrote commands"
			} else {
				summary += fmt.Sprintf(" 🚀 editing commands %s", m.buildSpinner.View())
			}
		}
		head := m.getBuildHeader(outputStatic)
		return style.Render(lipgloss.JoinVertical(lipgloss.Left, head, summary))
	}

	resRows := m.getRows(outputStatic)

	res := style.Render(strings.Join(resRows, "\n"))

	return res
}

func (m streamUIModel) didBuild() bool {
	return !(m.stopped || m.err != nil || m.apiErr != nil)
}

func (m streamUIModel) getBuildHeader(static bool) string {
	lbl := "Building plan "
	bgColor := color.BgGreen
	if static {
		if !m.didBuild() {
			lbl = "Build incomplete "
			bgColor = color.BgRed
		} else {
			lbl = "Built plan "
		}
	}

	head := color.New(bgColor, color.FgHiWhite, color.Bold).Sprint(" 🏗  ") + color.New(bgColor, color.FgHiWhite).Sprint(lbl)

	// Add collapse/expand hint
	var hint string
	if !static {
		hint = "(↓) collapse"
		if m.buildViewCollapsed {
			hint = "(↑) expand"
		}
	}
	padding := m.width - lipgloss.Width(head) - lipgloss.Width(hint) - 1 // 1 for space
	if padding > 0 {
		head += strings.Repeat(" ", padding) + hint
	}

	return head
}

func (m streamUIModel) getRows(static bool) []string {
	built := m.didBuild() && static
	head := m.getBuildHeader(static)

	// Gather file paths, _apply.sh last
	filePaths := make([]string, 0, len(m.tokensByPath))
	for filePath := range m.tokensByPath {
		if filePath == "_apply.sh" {
			continue
		}
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)
	if _, ok := m.tokensByPath["_apply.sh"]; ok {
		filePaths = append(filePaths, "_apply.sh")
	}

	var rows [][]string
	lineWidth := 0
	lineNum := -1
	rowIdx := 0

	for _, filePath := range filePaths {
		tokens := m.tokensByPath[filePath]
		finished := m.finished || m.finishedByPath[filePath] || built
		removed := m.removedByPath[filePath]

		// Basic block label
		icon := "📄"
		label := filePath
		if filePath == "_apply.sh" {
			icon = "🚀"
			label = "commands"
		}
		block := fmt.Sprintf("%s %s", icon, label)

		// Mark removed/finished/tokens
		switch {
		case removed:
			block += " ❌"
		case finished:
			block += " ✅"
		case tokens > 0:
			block += fmt.Sprintf(" %d 🪙", tokens)
		default:
			block += " " + m.buildSpinner.View()
		}

		// Truncate if needed
		blockWidth := lipgloss.Width(block)
		if blockWidth > m.width {
			maxWidth := m.width - lipgloss.Width("⋯")
			if maxWidth < 4 {
				block = string([]rune(block)[0:1]) + "⋯"
			} else {
				half := maxWidth / 2
				runes := []rune(block)
				block = string(runes[:half]) + "⋯" + string(runes[len(runes)-half:])
			}
		}

		// Build the "prefix + block" text tentatively:
		prefix := ""
		if rowIdx > 0 {
			prefix = " | "
		}
		candidate := prefix + block
		candidateWidth := lipgloss.Width(candidate)

		// Check if we have no row or it won't fit with the prefix
		if lineNum == -1 || lineWidth+candidateWidth > m.width {
			// Start a new row
			rows = append(rows, []string{})
			lineNum++
			rowIdx = 0
			lineWidth = 0

			// In a new row, there's no prefix
			candidate = block
			candidateWidth = lipgloss.Width(candidate)
		}

		rows[lineNum] = append(rows[lineNum], candidate)
		lineWidth += candidateWidth
		rowIdx++
	}

	// If empty row left at the end, strip it
	if len(rows) > 0 && len(rows[len(rows)-1]) == 0 {
		rows = rows[:len(rows)-1]
	}

	// Final output lines
	resRows := make([]string, len(rows)+1)
	resRows[0] = head
	for i, row := range rows {
		resRows[i+1] = lipgloss.JoinHorizontal(lipgloss.Left, row...)
	}

	return resRows
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
