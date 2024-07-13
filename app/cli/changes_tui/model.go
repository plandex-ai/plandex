package changes_tui

import (
	"github.com/charmbracelet/bubbles/help"
	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/plandex/plandex/shared"
)

type changesUIModel struct {
	help                     help.Model
	keymap                   keymap
	selectedFileIndex        int
	selectedReplacementIndex int
	selectedViewport         int
	currentPlan              *shared.CurrentPlanState
	changeOldViewport        viewport.Model
	changeNewViewport        viewport.Model
	fileViewport             viewport.Model
	selectionInfo            *selectionInfo
	ready                    bool
	width                    int
	height                   int
	shouldApplyAll           bool
	shouldRejectAll          bool
	didCopy                  bool
	isRejectingFile          bool
	isConfirmingRejectFile   bool
	rejectFileErr            *shared.ApiError
	justRejectedFile         bool
	spinner                  spinner.Model
}

type keymap = struct {
	up,
	down,
	left,
	right,
	scrollUp,
	scrollDown,
	pageUp,
	pageDown,
	start,
	end,
	switchView,
	reject,
	copy,
	applyAll,
	yes,
	no,
	quit bubbleKey.Binding
}

func (m changesUIModel) Init() tea.Cmd {
	return nil
}

func initialModel(currentPlan *shared.CurrentPlanState) *changesUIModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	initialState := changesUIModel{
		currentPlan:              currentPlan,
		selectedFileIndex:        0,
		selectedReplacementIndex: 0,
		help:                     help.New(),
		spinner:                  s,
		keymap: keymap{
			up: bubbleKey.NewBinding(
				bubbleKey.WithKeys("up"),
				bubbleKey.WithHelp("up", "prev change"),
			),

			down: bubbleKey.NewBinding(
				bubbleKey.WithKeys("down"),
				bubbleKey.WithHelp("down", "next change"),
			),
			left: bubbleKey.NewBinding(
				bubbleKey.WithKeys("left"),
				bubbleKey.WithHelp("left", "prev file"),
			),
			right: bubbleKey.NewBinding(
				bubbleKey.WithKeys("right"),
				bubbleKey.WithHelp("right", "next file"),
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

			start: bubbleKey.NewBinding(
				bubbleKey.WithKeys("g", "home"),
				bubbleKey.WithHelp("g", "start"),
			),

			end: bubbleKey.NewBinding(
				bubbleKey.WithKeys("G", "end"),
				bubbleKey.WithHelp("G", "end"),
			),

			switchView: bubbleKey.NewBinding(
				bubbleKey.WithKeys("tab"),
				bubbleKey.WithHelp("tab", "switch view"),
			),

			reject: bubbleKey.NewBinding(
				bubbleKey.WithKeys("r"),
				bubbleKey.WithHelp("r", "reject file"),
			),

			copy: bubbleKey.NewBinding(
				bubbleKey.WithKeys("c"),
				bubbleKey.WithHelp("c", "copy change"),
			),

			applyAll: bubbleKey.NewBinding(
				bubbleKey.WithKeys("ctrl+a"),
				bubbleKey.WithHelp("ctrl+a", "apply all changes"),
			),

			yes: bubbleKey.NewBinding(
				bubbleKey.WithKeys("y"),
				bubbleKey.WithHelp("y", "yes"),
			),

			no: bubbleKey.NewBinding(
				bubbleKey.WithKeys("n"),
				bubbleKey.WithHelp("n", "no"),
			),

			quit: bubbleKey.NewBinding(
				bubbleKey.WithKeys("q", "ctrl+c"),
				bubbleKey.WithHelp("q", "quit"),
			),
		},
	}

	return &initialState
}
