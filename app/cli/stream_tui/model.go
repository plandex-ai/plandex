package streamtui

import (
	"context"
	"log"
	"sync"
	"time"

	shared "plandex-shared"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

var stateMu sync.RWMutex

type streamUIModel struct {
	buildOnly   bool
	canSendToBg bool
	keymap      keymap

	reply       string
	mainDisplay string

	mainViewport viewport.Model

	processing   bool
	starting     bool
	spinner      spinner.Model
	buildSpinner spinner.Model
	sharedTicker *time.Ticker

	building       bool
	tokensByPath   map[string]int
	finishedByPath map[string]bool
	removedByPath  map[string]bool

	ready  bool
	width  int
	height int

	atScrollBottom bool

	promptingMissingFile   bool
	missingFilePath        string
	missingFileSelectedIdx int
	promptedMissingFile    bool
	autoLoadedMissingFile  bool
	missingFileContent     string
	missingFileTokens      int

	prompt string

	stopped    bool
	background bool
	finished   bool

	err    error
	apiErr *shared.ApiError

	updateDebouncer *UpdateDebouncer

	autoLoadContextCancelFn context.CancelFunc

	buildViewCollapsed bool
	userToggledBuild   bool
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
	background,
	enter bubbleKey.Binding
}

func (m streamUIModel) Init() tea.Cmd {
	log.Println("Model Init start")
	m.mainViewport.MouseWheelEnabled = true
	return tea.Batch(
		m.Tick(),
		m.pollBuildStatus(),
	)
}

type buildStatusPollMsg time.Time

func (m streamUIModel) pollBuildStatus() tea.Cmd {
	return tea.Every(5*time.Second, func(t time.Time) tea.Msg {
		return buildStatusPollMsg(t)
	})
}

func initialModel(prestartReply, prompt string, buildOnly bool, canSendToBg bool) *streamUIModel {
	sharedTicker := time.NewTicker(100 * time.Millisecond)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	buildSpinner := spinner.New()
	buildSpinner.Spinner = spinner.MiniDot

	initialState := streamUIModel{
		buildOnly:          buildOnly,
		canSendToBg:        canSendToBg,
		buildViewCollapsed: false,
		prompt:             prompt,
		reply:              prestartReply,
		keymap: keymap{
			quit: bubbleKey.NewBinding(
				bubbleKey.WithKeys("ctrl+c"),
				bubbleKey.WithHelp("ctrl+c", "quit"),
			),

			background: bubbleKey.NewBinding(
				bubbleKey.WithKeys("b"),
				bubbleKey.WithHelp("b", "background"),
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

		tokensByPath:    make(map[string]int),
		finishedByPath:  make(map[string]bool),
		removedByPath:   make(map[string]bool),
		spinner:         s,
		buildSpinner:    buildSpinner,
		sharedTicker:    sharedTicker,
		atScrollBottom:  true,
		starting:        true,
		updateDebouncer: NewUpdateDebouncer(8 * time.Millisecond),
	}

	return &initialState
}

func (m streamUIModel) Tick() tea.Cmd {
	return func() tea.Msg {
		<-m.sharedTicker.C
		return spinner.TickMsg{}
	}
}

func (m *streamUIModel) cleanup() {
	log.Println("Cleaning up stream UI model")
	m.updateState(func() {
		m.sharedTicker.Stop()
	})
}

func (m *streamUIModel) readState() streamUIModel {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return *m
}

func (m *streamUIModel) updateState(updateFn func()) {
	stateMu.Lock()
	defer stateMu.Unlock()
	updateFn()
}
