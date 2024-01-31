package streamtui

import (
	"plandex/term"
	"time"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/plandex/plandex/shared"
)

func (m *streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if msg.Type == tea.MouseWheelUp {
			m.replyViewport.LineUp(3)
		} else if msg.Type == tea.MouseWheelDown {
			m.replyViewport.LineDown(3)
		}

	case tea.KeyMsg:
		switch {
		case bubbleKey.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case bubbleKey.Matches(msg, m.keymap.scrollDown):
			m.scrollDown()
		case bubbleKey.Matches(msg, m.keymap.scrollUp):
			m.scrollUp()
		case bubbleKey.Matches(msg, m.keymap.pageDown):
			m.pageDown()
		case bubbleKey.Matches(msg, m.keymap.pageUp):
			m.pageUp()
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
	switch msg.Type {
	case shared.StreamMessageReply:
		if m.starting {
			m.starting = false
		}

		if m.processing {
			m.processing = false
			m.reply += "\n\n👉 "
		}

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
