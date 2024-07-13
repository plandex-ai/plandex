package changes_tui

import (
	"plandex/types"
	"time"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/plandex/plandex/shared"
)

type toggleDidCopyOffMsg struct{}
type finishedRejectFile struct {
	planState *shared.CurrentPlanState
	err       *shared.ApiError
}

func (m changesUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Println("msg:", msg)

	if m.selectionInfo == nil {
		m.setSelectionInfo()
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.windowResized(msg.Width, msg.Height)

	case types.ChangesUIViewportsUpdate:
		m.updateViewportSizes()
		if msg.ScrollReplacement != nil {
			m.scrollReplacementIntoView(msg.ScrollReplacement.OldContent, msg.ScrollReplacement.NewContent, msg.ScrollReplacement.NumLinesPrepended)
		}

	case spinner.TickMsg:
		if m.isRejectingFile {
			spinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = spinnerModel
			return m, cmd
		}

	case toggleDidCopyOffMsg:
		m.didCopy = false

	case finishedRejectFile:
		m.justRejectedFile = true

		if msg.err != nil {
			m.rejectFileErr = msg.err
			return m, tea.Quit
		}

		m.currentPlan = msg.planState

		if len(msg.planState.PlanResult.SortedPaths) == 0 {
			return m, tea.Quit
		}

		m.isRejectingFile = false
		m.justRejectedFile = false
		m.selectedFileIndex = 0
		m.selectedReplacementIndex = 0
		m.setSelectionInfo()
		m.updateMainView(true)

	case tea.KeyMsg:
		if m.isConfirmingRejectFile {
			if !bubbleKey.Matches(msg, m.keymap.yes) && !bubbleKey.Matches(msg, m.keymap.no) &&
				!bubbleKey.Matches(msg, m.keymap.quit) {
				return m, nil
			}
		}

		if m.isRejectingFile {
			if !bubbleKey.Matches(msg, m.keymap.quit) {
				return m, nil
			}
		}

		switch {

		case bubbleKey.Matches(msg, m.keymap.left):
			m.left()

		case bubbleKey.Matches(msg, m.keymap.right):
			m.right()

		case bubbleKey.Matches(msg, m.keymap.up):
			m.up()

		case bubbleKey.Matches(msg, m.keymap.down):
			m.down()

		case bubbleKey.Matches(msg, m.keymap.scrollDown):
			m.scrollDown()

		case bubbleKey.Matches(msg, m.keymap.scrollUp):
			m.scrollUp()

		case bubbleKey.Matches(msg, m.keymap.pageDown):
			m.pageDown()

		case bubbleKey.Matches(msg, m.keymap.pageUp):
			m.pageUp()

		case bubbleKey.Matches(msg, m.keymap.start):
			m.start()

		case bubbleKey.Matches(msg, m.keymap.end):
			m.end()

		case bubbleKey.Matches(msg, m.keymap.switchView):
			m.switchView()

		case bubbleKey.Matches(msg, m.keymap.reject):
			m.isConfirmingRejectFile = true

		case bubbleKey.Matches(msg, m.keymap.yes):
			m.isRejectingFile = true
			m.isConfirmingRejectFile = false
			started := time.Now()
			go func() {
				planState, err := m.rejectFile()

				// If the rejectFile call took less than 500ms, wait a bit so the spinner
				// doesn't flash jarringly.
				elapsed := time.Since(started)
				if elapsed < 500*time.Millisecond {
					time.Sleep((500 * time.Millisecond) - elapsed)
				}

				if err != nil {
					program.Send(finishedRejectFile{err: err})
					return
				}
				program.Send(finishedRejectFile{planState: planState})
			}()
			return m, m.spinner.Tick

		case bubbleKey.Matches(msg, m.keymap.no):
			m.isConfirmingRejectFile = false

		case bubbleKey.Matches(msg, m.keymap.applyAll):
			m.shouldApplyAll = true
			return m, tea.Quit

		case bubbleKey.Matches(msg, m.keymap.copy):
			m.copyCurrentChange()
			time.AfterFunc(1*time.Second, func() {
				program.Send(toggleDidCopyOffMsg{})
			})

		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit

		default:
			// handle escape sequences sometimes sent by arrow keys
			m.resolveEscapeSequence(msg.String())
		}
	}

	return m, nil
}

var escReceivedAt time.Time
var escSeq string

func (m *changesUIModel) resolveEscapeSequence(val string) {
	if val == "esc" || val == "alt+[" {
		escReceivedAt = time.Now()
		go func() {
			time.Sleep(51 * time.Millisecond)
			escReceivedAt = time.Time{}
			escSeq = ""
		}()
	}

	if !escReceivedAt.IsZero() {
		elapsed := time.Since(escReceivedAt)

		if elapsed < 50*time.Millisecond {
			escSeq += val

			if escSeq == "esc[A" || escSeq == "alt+[A" {
				m.up()
				escReceivedAt = time.Time{}
				escSeq = ""
			} else if escSeq == "esc[B" || escSeq == "alt+[B" {
				m.down()
				escReceivedAt = time.Time{}
				escSeq = ""
			} else if escSeq == "esc[C" || escSeq == "alt+[C" {
				m.right()
				escReceivedAt = time.Time{}
				escSeq = ""
			} else if escSeq == "esc[D" || escSeq == "alt+[D" {
				m.left()
				escReceivedAt = time.Time{}
				escSeq = ""
			}
		}
	}
}
