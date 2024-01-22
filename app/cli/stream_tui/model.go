package streamtui

import (
	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type streamUIModel struct {
	keymap keymap

	reply        string
	replyDisplay string

	replyViewport viewport.Model

	processing bool
	spinner    spinner.Model

	building       bool
	tokensByPath   map[string]int
	finishedByPath map[string]bool

	ready  bool
	width  int
	height int

	atScrollBottom bool
}

type keymap = struct {
	stop,
	scrollUp,
	scrollDown,
	pageUp,
	pageDown,
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
		},

		tokensByPath:   make(map[string]int),
		finishedByPath: make(map[string]bool),
		spinner:        s,
		atScrollBottom: true,
	}

	return &initialState
}
