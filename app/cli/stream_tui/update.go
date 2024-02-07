package streamtui

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/lib"
	"plandex/term"
	"strings"
	"time"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/plandex/plandex/shared"
)

func (m streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Println("Update received message:", spew.Sdump(msg))

	switch msg := msg.(type) {

	case spinner.TickMsg:
		if m.processing || m.starting {
			spinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = spinnerModel
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.windowResized(msg.Width, msg.Height)

	case shared.StreamMessage:
		return m.streamUpdate(&msg)

	case delayFileRestartMsg:
		m.finishedByPath[msg.path] = false

	case tea.MouseMsg:
		if !m.promptingMissingFile {
			if msg.Type == tea.MouseWheelUp {
				m.replyViewport.LineUp(3)
			} else if msg.Type == tea.MouseWheelDown {
				m.replyViewport.LineDown(3)
			}
		}

	case tea.KeyMsg:
		switch {
		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case bubbleKey.Matches(msg, m.keymap.scrollDown) && !m.promptingMissingFile:
			m.scrollDown()
		case bubbleKey.Matches(msg, m.keymap.scrollUp) && !m.promptingMissingFile:
			m.scrollUp()
		case bubbleKey.Matches(msg, m.keymap.pageDown) && !m.promptingMissingFile:
			m.pageDown()
		case bubbleKey.Matches(msg, m.keymap.pageUp) && !m.promptingMissingFile:
			m.pageUp()
		case bubbleKey.Matches(msg, m.keymap.up):
			m.up()
		case bubbleKey.Matches(msg, m.keymap.down):
			m.down()
		case m.promptingMissingFile && bubbleKey.Matches(msg, m.keymap.enter):
			m.selectedMissingFileOpt()

		default:
			m.resolveEscapeSequence(msg.String())
		}
	}

	return m, nil
}

func (m *streamUIModel) windowResized(w, h int) {
	m.width = w
	m.height = h

	_, viewportHeight := m.getViewportDimensions()

	if m.ready {
		m.updateViewportDimensions()
	} else {
		m.replyViewport = viewport.New(w, viewportHeight)
		m.replyViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
		m.ready = true
	}
}

func (m *streamUIModel) updateReplyDisplay() {
	md, _ := term.GetMarkdown(m.reply)
	m.replyDisplay = md
	m.replyViewport.SetContent(md)
	m.updateViewportDimensions()

	if m.atScrollBottom {
		m.replyViewport.GotoBottom()
	}
}

func (m *streamUIModel) updateViewportDimensions() {
	w, h := m.getViewportDimensions()
	m.replyViewport.Width = w
	m.replyViewport.Height = h
}

func (m *streamUIModel) getViewportDimensions() (int, int) {
	w := m.width
	h := m.height

	helpHeight := lipgloss.Height(m.renderHelp())
	buildHeight := lipgloss.Height(m.renderBuild())
	processingHeight := lipgloss.Height(m.renderProcessing())
	maxViewportHeight := h - (helpHeight + processingHeight + buildHeight)
	viewportHeight := min(maxViewportHeight, lipgloss.Height(m.replyDisplay))
	viewportWidth := w

	// log.Println("viewportWidth:", viewportWidth)

	return viewportWidth, viewportHeight
}

func (m streamUIModel) replyScrollable() bool {
	return m.replyViewport.TotalLineCount() > m.replyViewport.VisibleLineCount()
}

func (m *streamUIModel) scrollDown() {
	if m.replyScrollable() {
		m.replyViewport.LineDown(1)
	}

	m.atScrollBottom = !m.replyScrollable() || m.replyViewport.AtBottom()
}

func (m *streamUIModel) scrollUp() {
	if m.replyScrollable() {
		m.replyViewport.LineUp(1)
		m.atScrollBottom = false
	}
}

func (m *streamUIModel) pageDown() {
	if m.replyScrollable() {
		m.replyViewport.ViewDown()
	}

	m.atScrollBottom = !m.replyScrollable() || m.replyViewport.AtBottom()
}

func (m *streamUIModel) pageUp() {
	if m.replyScrollable() {
		m.replyViewport.ViewUp()
		m.atScrollBottom = false
	}
}

func (m *streamUIModel) streamUpdate(msg *shared.StreamMessage) (tea.Model, tea.Cmd) {
	// log.Println("streamUI received message:", msg.Type)
	switch msg.Type {

	case shared.StreamMessagePromptMissingFile:
		m.promptingMissingFile = true
		m.missingFilePath = msg.MissingFilePath

		bytes, err := os.ReadFile(m.missingFilePath)
		if err != nil {
			log.Println("failed to read file:", err)
			m.err = fmt.Errorf("failed to read file: %w", err)
			return m, nil
		}
		m.missingFileContent = string(bytes)

		numTokens, err := shared.GetNumTokens(m.missingFileContent)

		if err != nil {
			log.Println("failed to get num tokens:", err)
			m.err = fmt.Errorf("failed to get num tokens: %w", err)
			return m, nil
		}

		m.missingFileTokens = numTokens

	case shared.StreamMessageReply:
		if m.starting {
			m.starting = false
		}

		if m.processing {
			m.processing = false
			if m.promptedMissingFile {
				m.promptedMissingFile = false
			} else {
				m.reply += "\n\n👉 "
			}
		}

		// log.Println("reply chunk:", msg.ReplyChunk)

		m.reply += msg.ReplyChunk
		m.updateReplyDisplay()

	case shared.StreamMessageBuildInfo:
		m.building = true
		wasFinished := m.finishedByPath[msg.BuildInfo.Path]
		nowFinished := msg.BuildInfo.Finished

		if msg.BuildInfo.Finished {
			m.tokensByPath[msg.BuildInfo.Path] = 0
			m.finishedByPath[msg.BuildInfo.Path] = true
		} else {
			if wasFinished && !nowFinished {
				// delay for a second before marking not finished again (so check flashes green prior to restarting build)
				return m, startDelay(msg.BuildInfo.Path, time.Second*1)
			} else {
				m.finishedByPath[msg.BuildInfo.Path] = false
			}

			m.tokensByPath[msg.BuildInfo.Path] += msg.BuildInfo.NumTokens
		}

		m.updateViewportDimensions()

	case shared.StreamMessageDescribing:
		m.processing = true
		return m, m.spinner.Tick

	case shared.StreamMessageError:

	case shared.StreamMessageFinished:

	case shared.StreamMessageAborted:
	}

	return m, nil
}

type delayFileRestartMsg struct {
	path string
}

func startDelay(path string, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return delayFileRestartMsg{path: path}
	}
}

var escReceivedAt time.Time
var escSeq string

func (m *streamUIModel) resolveEscapeSequence(val string) {
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
				// log.Println("up")
				m.up()

				escReceivedAt = time.Time{}
				escSeq = ""
			} else if escSeq == "esc[B" || escSeq == "alt+[B" {
				// log.Println("down")
				m.down()

				escReceivedAt = time.Time{}
				escSeq = ""
			}
		}
	}
}

func (m *streamUIModel) up() {
	if m.promptingMissingFile {
		m.missingFileSelectedIdx = max(m.missingFileSelectedIdx-1, 0)
	}
}

func (m *streamUIModel) down() {
	if m.promptingMissingFile {
		m.missingFileSelectedIdx = min(m.missingFileSelectedIdx+1, len(missingFileSelectOpts)-1)
	}

}

func (m *streamUIModel) selectedMissingFileOpt() {
	choice := promptChoices[m.missingFileSelectedIdx]

	if choice == "" {
		return
	}

	apiErr := api.Client.RespondMissingFile(lib.CurrentPlanId, lib.CurrentBranch, shared.RespondMissingFileRequest{
		Choice:   choice,
		FilePath: m.missingFilePath,
		Body:     m.missingFileContent,
	})

	if apiErr != nil {
		log.Println("missing file prompt api error:", apiErr)
		m.apiErr = apiErr
		return
	}

	if choice == shared.RespondMissingFileChoiceSkip {
		replyLines := strings.Split(m.reply, "\n")
		m.reply = strings.Join(replyLines[:len(replyLines)-3], "\n")
		m.updateReplyDisplay()
	}

	m.promptingMissingFile = false
	m.missingFilePath = ""
	m.missingFileSelectedIdx = 0
	m.missingFileContent = ""
	m.missingFileTokens = 0
	m.promptedMissingFile = true
	m.processing = true
}
