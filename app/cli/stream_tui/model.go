package streamtui

import (
	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/plandex/plandex/shared"
)

const (
	MissingFileLoadLabel      = "Load the file into context"
	MissingFileSkipLabel      = "Skip generating this file"
	MissingFileOverwriteLabel = "Allow Plandex to overwrite this file"
)

var promptChoices = []shared.RespondMissingFileChoice{
	shared.RespondMissingFileChoiceLoad,
	shared.RespondMissingFileChoiceSkip,
	shared.RespondMissingFileChoiceOverwrite,
}

var missingFileSelectOpts = []string{
	MissingFileLoadLabel,
	MissingFileSkipLabel,
	MissingFileOverwriteLabel,
}

type streamUIModel struct {
	buildOnly bool
	keymap    keymap

	reply       string
	mainDisplay string

	mainViewport viewport.Model

	processing bool
	starting   bool
	spinner    spinner.Model

	building       bool
	tokensByPath   map[string]int
	finishedByPath map[string]bool

	ready  bool
	width  int
	height int

	atScrollBottom bool

	promptingMissingFile   bool
	missingFilePath        string
	missingFileSelectedIdx int
	promptedMissingFile    bool
	missingFileContent     string
	missingFileTokens      int

	prompt string

	stopped    bool
	background bool
	finished   bool

	err    error
	apiErr *shared.ApiError
}

type keymap = struct {
	stop,
	scrollUp,
	scrollDown,
	pageUp,
	pageDown,
	start,
	end,
	up,
	down,
	quit,
	enter bubbleKey.Binding
}

func (m streamUIModel) Init() tea.Cmd {
	m.mainViewport.MouseWheelEnabled = true

	// start spinner
	return m.spinner.Tick
}

func initialModel(prestartReply, prompt string, buildOnly bool) *streamUIModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	initialState := streamUIModel{
		buildOnly: buildOnly,
		prompt:    prompt,
		reply:     prestartReply,
		keymap: keymap{
			quit: bubbleKey.NewBinding(
				bubbleKey.WithKeys("b", "ctrl+c"),
				bubbleKey.WithHelp("ctrl+c", "quit"),
			),

			stop: bubbleKey.NewBinding(
				bubbleKey.WithKeys("s"),
				bubbleKey.WithHelp("s", "stop"),
			),

			scrollDown: bubbleKey.NewBinding(
				bubbleKey.WithKeys("j"),
				bubbleKey.WithHelp("j", "scroll down"),
			),

			scrollUp: bubbleKey.NewBinding(
				bubbleKey.WithKeys("k"),
				bubbleKey.WithHelp("k", "scroll up"),
			),

			pageDown: bubbleKey.NewBinding(
				bubbleKey.WithKeys("d", "pageDown"),
				bubbleKey.WithHelp("d", "page down"),
			),

			pageUp: bubbleKey.NewBinding(
				bubbleKey.WithKeys("u", "pageUp"),
				bubbleKey.WithHelp("u", "page up"),
			),

			up: bubbleKey.NewBinding(
				bubbleKey.WithKeys("up"),
				bubbleKey.WithHelp("up", "prev"),
			),

			down: bubbleKey.NewBinding(
				bubbleKey.WithKeys("down"),
				bubbleKey.WithHelp("down", "next"),
			),

			enter: bubbleKey.NewBinding(
				bubbleKey.WithKeys("enter"),
				bubbleKey.WithHelp("enter", "select"),
			),

			start: bubbleKey.NewBinding(
				bubbleKey.WithKeys("g", "home"),
				bubbleKey.WithHelp("g", "start"),
			),

			end: bubbleKey.NewBinding(
				bubbleKey.WithKeys("G", "end"),
				bubbleKey.WithHelp("G", "end"),
			),
		},

		tokensByPath:   make(map[string]int),
		finishedByPath: make(map[string]bool),
		spinner:        s,
		atScrollBottom: true,
		starting:       true,
	}

	return &initialState
}
