package changes_tui

import (
	"fmt"
	"log"
	"plandex/api"
	"plandex/lib"

	"github.com/charmbracelet/bubbles/help"
	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	switchView,
	reject,
	copy,
	applyAll,
	rejectAll,
	quit bubbleKey.Binding
}

func (m changesUIModel) Init() tea.Cmd {
	return nil
}

func initialModel() *changesUIModel {
	currentPlan, apiErr := api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)
	if apiErr != nil {
		err := fmt.Errorf("error getting current plan state: %v", apiErr.Msg)
		log.Println(err)
		panic(err)
	}

	initialState := changesUIModel{
		currentPlan:              currentPlan,
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

			switchView: bubbleKey.NewBinding(
				bubbleKey.WithKeys("tab"),
				bubbleKey.WithHelp("tab", "switch view"),
			),

			reject: bubbleKey.NewBinding(
				bubbleKey.WithKeys("d"),
				bubbleKey.WithHelp("d", "drop change"),
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

	return &initialState
}
