package changes_tui

import (
	"fmt"
	"log"
	"plandex/lib"
	"plandex/types"

	"github.com/charmbracelet/bubbles/help"
	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type changesUIModel struct {
	help                     help.Model
	keymap                   keymap
	selectedFileIndex        int
	selectedReplacementIndex int
	selectedViewport         int
	resultsInfo              *types.PlanResultsInfo
	currentPlan              *types.CurrentPlanState
	changeOldViewport        viewport.Model
	changeNewViewport        viewport.Model
	fileViewport             viewport.Model
	selectionInfo            *selectionInfo
	ready                    bool
	width                    int
	height                   int
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
	currentPlan, err := lib.GetCurrentPlanState()
	if err != nil {
		err = fmt.Errorf("error getting current plan state: %v", err)
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

			scrollDown: bubbleKey.NewBinding(
				bubbleKey.WithKeys("j"),
				bubbleKey.WithHelp("j", "scroll down"),
			),

			scrollUp: bubbleKey.NewBinding(
				bubbleKey.WithKeys("k"),
				bubbleKey.WithHelp("k", "scroll up"),
			),

			switchView: bubbleKey.NewBinding(
				bubbleKey.WithKeys("tab"),
				bubbleKey.WithHelp("tab", "switch view"),
			),

			reject: bubbleKey.NewBinding(
				bubbleKey.WithKeys("d"),
				bubbleKey.WithHelp("d", "discard change"),
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
