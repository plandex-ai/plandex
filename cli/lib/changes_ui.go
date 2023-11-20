package lib

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"

	"github.com/atotto/clipboard"
)

type keymap = struct {
	up, down, reject, copy, applyAll, quit bubbleKey.Binding
}

type changesUIModel struct {
	help                     help.Model
	keymap                   keymap
	selectedFileIndex        int
	selectedReplacementIndex int
	resultsInfo              *planResultsInfo
	currentPlanFiles         *shared.CurrentPlanFiles
	sidebar                  list.Model
}

func StartChangesUI() error {
	log.Print("\n\n\nStarting changes UI...\n\n")

	_, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run()

	if err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}

	// fmt.Println("exited changes UI")
	// spew.Dump(m)

	return nil
}

func (m changesUIModel) Init() tea.Cmd {
	return nil
}

func initialModel() changesUIModel {
	errChn := make(chan error, 1)
	resultsInfoCh := make(chan *planResultsInfo, 1)
	currentPlanFilesCh := make(chan *shared.CurrentPlanFiles, 1)

	var resultsInfo *planResultsInfo
	var currentPlanFiles *shared.CurrentPlanFiles

	go func() {
		planResByPath, err := getPlanResultsInfo()
		if err != nil {
			errChn <- fmt.Errorf("error getting plan results: %v", err)
			return
		}
		resultsInfoCh <- planResByPath
	}()

	go func() {
		var err error
		currentPlanFiles, err = GetCurrentPlanFiles()
		if err != nil {
			errChn <- fmt.Errorf("error getting current plan files: %v", err)
			return
		}
		currentPlanFilesCh <- currentPlanFiles
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChn:
			panic(fmt.Errorf("error getting plan results: %v", err))
		case resultsInfo = <-resultsInfoCh:
		case currentPlanFiles = <-currentPlanFilesCh:
		}
	}

	initialState := changesUIModel{
		resultsInfo:              resultsInfo,
		selectedFileIndex:        0,
		selectedReplacementIndex: 0,
		help:                     help.New(),
		keymap: keymap{
			up: bubbleKey.NewBinding(
				bubbleKey.WithKeys("up"),
				bubbleKey.WithHelp("up", "prev change"),
			),

			down: bubbleKey.NewBinding(
				bubbleKey.WithKeys("down"),
				bubbleKey.WithHelp("down", "next change"),
			),

			reject: bubbleKey.NewBinding(
				bubbleKey.WithKeys("r"),
				bubbleKey.WithHelp("r", "reject change"),
			),

			copy: bubbleKey.NewBinding(
				bubbleKey.WithKeys("c"),
				bubbleKey.WithHelp("c", "copy change"),
			),

			applyAll: bubbleKey.NewBinding(
				bubbleKey.WithKeys("ctrl+a"),
				bubbleKey.WithHelp("ctrl+a", "apply all changes"),
			),

			quit: bubbleKey.NewBinding(
				bubbleKey.WithKeys("q", "ctrl+c"),
				bubbleKey.WithHelp("q", "quit"),
			),
		},
	}

	return initialState
}

var escPressedAt time.Time
var escSeq string

func (m changesUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	selectionInfo := m.getSelectionInfo()
	paths := m.resultsInfo.sortedPaths
	currentReplacements := selectionInfo.currentReplacements

	log.Printf("msg: %v", msg)

	up := func() {
		log.Println("up")
		log.Printf("selectedReplacementIndex before: %d", m.selectedReplacementIndex)
		log.Printf("selectedFileIndex before: %d", m.selectedFileIndex)
		if m.selectedReplacementIndex > 0 {
			m.selectedReplacementIndex--
		} else if m.selectedFileIndex > 0 {
			m.selectedFileIndex--
			m.selectedReplacementIndex = len(m.resultsInfo.replacementsByPath[paths[m.selectedFileIndex]]) - 1
		}

		log.Printf("selectedReplacementIndex after: %d", m.selectedReplacementIndex)
		log.Printf("selectedFileIndex after: %d", m.selectedFileIndex)
	}

	down := func() {
		log.Println("down")
		log.Printf("selectedReplacementIndex before: %d", m.selectedReplacementIndex)
		log.Printf("len(currentReplacements)-1: %d", len(currentReplacements)-1)
		log.Printf("selectedFileIndex before: %d", m.selectedFileIndex)
		log.Printf("len(paths)-1: %d", len(paths)-1)
		if m.selectedReplacementIndex < len(currentReplacements)-1 {
			m.selectedReplacementIndex++
		} else if m.selectedFileIndex < len(paths)-1 {
			m.selectedFileIndex++
			m.selectedReplacementIndex = 0
		}
		log.Printf("selectedReplacementIndex after: %d", m.selectedReplacementIndex)
		log.Printf("selectedFileIndex after: %d", m.selectedFileIndex)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch {

		case bubbleKey.Matches(msg, m.keymap.up):
			up()

		case bubbleKey.Matches(msg, m.keymap.down):
			down()

		case bubbleKey.Matches(msg, m.keymap.reject):
			m.rejectChange()

		case bubbleKey.Matches(msg, m.keymap.applyAll):
			// TODO: need to close the TUI then apply

		case bubbleKey.Matches(msg, m.keymap.copy):
			m.copyCurrentChange()

		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit

		default:
			if msg.String() == "esc" {
				// log.Println("esc")
				escPressedAt = time.Now()
				go func() {
					time.Sleep(51 * time.Millisecond)
					// log.Println("esc timeout")
					escPressedAt = time.Time{}
					escSeq = ""
				}()
			}

			if !escPressedAt.IsZero() {
				elapsed := time.Since(escPressedAt)

				// log.Println("elapsed: ", elapsed)

				if elapsed < 50*time.Millisecond {
					escSeq += msg.String()

					// log.Println("escSeq: ", escSeq)

					if escSeq == "esc[A" {
						up()
						escPressedAt = time.Time{}
						escSeq = ""
					} else if escSeq == "esc[B" {
						down()
						escPressedAt = time.Time{}
						escSeq = ""
					}

					return m, nil
				}
			}

		}
	}

	return m, nil
}

func (m changesUIModel) View() string {
	help := m.help.ShortHelpView([]bubbleKey.Binding{
		m.keymap.up,
		m.keymap.down,
		m.keymap.applyAll,
		m.keymap.quit,
	})

	sidebar := m.renderSidebar()
	mainView := m.renderMainView()

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainView)

	return lipgloss.JoinVertical(lipgloss.Left,
		layout,
		help,
	)
}

func (m *changesUIModel) renderSidebar() string {
	// log.Print("renderSidebar")

	var sb strings.Builder

	selectionInfo := m.getSelectionInfo()
	paths := m.resultsInfo.sortedPaths
	currentPath := selectionInfo.currentPath
	currentRes := selectionInfo.currentRes
	currentRep := selectionInfo.currentRep

	// log.Printf("paths: %v", paths)
	// log.Printf("currentPath: %v", currentPath)
	// log.Printf("currentRes: %v", currentRes)
	// log.Printf("currentRep: %v", currentRep)

	for _, path := range paths {
		results := m.resultsInfo.resultsByPath[path]
		selectedPath := path == currentPath

		selectedFullFile := selectedPath && currentRes == nil

		pathColor := color.FgHiGreen

		anyFailed := false
		for _, result := range results {
			if result.AnyFailed {
				anyFailed = true
				break
			}
		}
		if anyFailed {
			pathColor = color.FgHiRed
		}

		sb.WriteString(color.New(pathColor).Sprint(" 📄 " + path + "\n"))

		// Change entries
		for _, result := range results {
			for _, rep := range result.Replacements {
				selected := selectedPath && rep.Id == currentRep.Id

				s := ""

				labelColor := color.FgHiGreen
				if rep.Failed {
					labelColor = color.FgHiRed
				}

				if selected {
					s += color.New(color.Bold).Sprintf(" > ") + color.New(color.Bold, labelColor).Sprint(shortenText(rep.Summary, 50))
				} else {
					s += color.New(labelColor).Sprintf(" - %s", shortenText(rep.Summary, 50))
				}

				if result.RejectedAt != "" {
					s += " 🚫"
				}

				s += "\n"

				sb.WriteString(s)
			}

		}

		if selectedFullFile {
			sb.WriteString(color.New(color.Bold, color.FgHiCyan).Sprint(" > ") + color.New(color.Bold).Sprint("Updated file\n"))
		} else {
			sb.WriteString(" - Updated file\n")
		}
	}

	return sb.String()
}

// Helper function to shorten text if necessary
func shortenText(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

func (m *changesUIModel) renderMainView() string {
	selectionInfo := m.getSelectionInfo()

	if selectionInfo.currentRep == nil {
		// Show whole file

	} else {
		// Show replacement
		return fmt.Sprintf(`
		Old:

		`+"```"+`
		%s
		`+"```"+`

		New:

		`+"```"+`
		%s
		`+"```", selectionInfo.currentRep.Old, selectionInfo.currentRep.New)
	}

	return ""
}

func (m *changesUIModel) rejectChange() error {

	return nil
}

func (m *changesUIModel) applyAllChanges() error {

	return nil
}

type selectionInfo struct {
	currentPath         string
	currentRes          *shared.PlanResult
	currentReplacements []*shared.Replacement
	currentRep          *shared.Replacement
}

func (m *changesUIModel) getSelectionInfo() selectionInfo {
	paths := m.resultsInfo.sortedPaths
	currentPath := paths[m.selectedFileIndex]

	results := m.resultsInfo.resultsByPath[currentPath]

	var currentRes *shared.PlanResult
	var currentRep *shared.Replacement

	var pathReplacements []*shared.Replacement

	for i, res := range results {
		for j, rep := range res.Replacements {
			pathReplacements = append(pathReplacements, rep)

			flatIndex := i*len(res.Replacements) + j
			if flatIndex == m.selectedReplacementIndex {
				currentRes = res
				currentRep = rep
			}
		}
	}

	return selectionInfo{
		currentPath:         currentPath,
		currentRes:          currentRes,
		currentReplacements: pathReplacements,
		currentRep:          currentRep,
	}
}

func (m *changesUIModel) copyCurrentChange() error {
	selectionInfo := m.getSelectionInfo()
	if selectionInfo.currentRep == nil {
		return fmt.Errorf("no change is currently selected")
	}

	// Copy the 'New' content of the replacement to the clipboard
	if err := clipboard.WriteAll(selectionInfo.currentRep.New); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}

	return nil
}
