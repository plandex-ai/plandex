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
	keymap keymap

	reply        string
	replyDisplay string

	replyViewport viewport.Model

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

	err    error
	apiErr *shared.ApiError
}

type keymap = struct {
	stop,
	scrollUp,
	scrollDown,
	pageUp,
	pageDown,
	up,
	down,
	enter,
	quit bubbleKey.Binding
}

func (m streamUIModel) Init() tea.Cmd {
	m.replyViewport.MouseWheelEnabled = true

	// start spinner
	return m.spinner.Tick
}

func initialModel() *streamUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	initialState := streamUIModel{
		keymap: keymap{
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
				bubbleKey.WithKeys("J", "pageDown"),
				bubbleKey.WithHelp("J", "page down"),
			),

			pageUp: bubbleKey.NewBinding(
				bubbleKey.WithKeys("K", "pageUp"),
				bubbleKey.WithHelp("K", "page up"),
			),
			quit: bubbleKey.NewBinding(
				bubbleKey.WithKeys("ctrl+c"),
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
		},

		tokensByPath:   make(map[string]int),
		finishedByPath: make(map[string]bool),
		spinner:        s,
		atScrollBottom: true,
		starting:       true,
	}

	return &initialState
}
