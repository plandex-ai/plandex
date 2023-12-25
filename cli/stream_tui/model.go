package streamtui

import (
	"time"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type streamUIModel struct {
	keymap keymap

	reply                 string
	replyDisplay          string
	replyDisplayUpdatedAt time.Time

	replyViewport viewport.Model

	processing bool
	spinner    spinner.Model

	building       bool
	tokensByPath   map[string]int
	finishedByPath map[string]bool

	ready  bool
	width  int
	height int
}

type keymap = struct {
	quit bubbleKey.Binding
}

func (m streamUIModel) Init() tea.Cmd {
	return nil
}

func initialModel() *streamUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	initialState := streamUIModel{
		keymap: keymap{
			quit: bubbleKey.NewBinding(
				bubbleKey.WithKeys("q", "ctrl+c"),
				bubbleKey.WithHelp("q", "quit"),
			),
		},

		tokensByPath:   make(map[string]int),
		finishedByPath: make(map[string]bool),
		spinner:        s,
	}

	return &initialState
}
