package lib

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	bubbleKey "github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/plandex/plandex/shared"
)

type keymap = struct {
	up, down, reject, copy, applyAll, quit bubbleKey.Binding
}

type UIState int

const (
	ViewState UIState = iota
	ChangeState
)

type changeDetail struct {
	Path      string
	Index     int
	Change    shared.PlanResultReplacement
	OldView   string
	NewView   string
}

type changesUIModel struct {
	help                     help.Model
	keymap                   keymap
	selectedFileIndex        int
	selectedReplacementIndex int
	resultsByPath            shared.PlanResultsByPath
	currentPlanFiles         *shared.CurrentPlanFiles
	state                    UIState
	fileViewer               string
	changeDetails            *changeDetail
	clipboard                clipboardUtil
}

type clipboardUtil struct{}

func (c *clipboardUtil) copy(text string) error {
	// Implementation goes here.
	return nil
}

func StartChangesUI() error {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("error running changes UI: %v", err)
	}
	return nil
}

func (m changesUIModel) Init() tea.Cmd {
	return nil
}

func initialModel() changesUIModel {
	errChn := make(chan error, 1)
	planResByPathChn := make(chan shared.PlanResultsByPath, 1)
	currentPlanFilesChn := make(chan *shared.CurrentPlanFiles, 1)

	var resultsByPath shared.PlanResultsByPath
	var currentPlanFiles *shared.CurrentPlanFiles

	go func() {
		planResByPath, err := getPlanResultsByPath()
		if err != nil {
			errChn <- fmt.Errorf("error getting plan results: %v", err)
			return
		}
		planResByPathChn <- planResByPath
	}()

	go func() {
		var err error
		currentPlanFiles, err = GetCurrentPlanFiles()
		if err != nil {
			errChn <- fmt.Errorf("error getting current plan files: %v", err)
			return
		}
		currentPlanFilesChn <- currentPlanFiles
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChn:
			panic(fmt.Errorf("error getting plan results: %v", err))
		case resultsByPath = <-planResByPathChn:
		case currentPlanFiles = <-currentPlanFilesChn:
		}
	}

	state:  ViewState,
		fileViewer:    resultsByPath["example.go"][0].Content,
		changeDetails: &changeDetail{
			Path: "example.go",
			Index: 0,
		},
		clipboard:     clipboardUtil{},
		resultsByPath: resultsByPath,
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
				bubbleKey.WithKeys("esc"),
				bubbleKey.WithHelp("esc", "quit"),
			),
		},
	}
}

func (m changesUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch {

		case bubbleKey.Matches(msg, m.keymap.up):

		case bubbleKey.Matches(msg, m.keymap.down):

		case bubbleKey.Matches(msg, m.keymap.reject):

		case bubbleKey.Matches(msg, m.keymap.applyAll):

		case bubbleKey.Matches(msg, m.keymap.copy):

		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit

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

	var views []string

	return lipgloss.JoinHorizontal(lipgloss.Top, views...) + "\n\n" + help
}
