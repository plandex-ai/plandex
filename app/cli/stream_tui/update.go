package streamtui

import (
	"plandex/term"

	bubbleKey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/plandex/plandex/shared"
)

func (m *streamUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		if m.processing {
			spinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = spinnerModel
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.windowResized(msg.Width, msg.Height)

	case shared.StreamMessage:
		return m.streamUpdate(&msg)

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
	maxViewportHeight := h - (helpHeight + buildHeight + processingHeight)
	viewportHeight := min(maxViewportHeight, lipgloss.Height(m.replyDisplay))

	return w, viewportHeight
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
		m.processing = false
		m.reply += msg.ReplyChunk
		m.updateReplyDisplay()

	case shared.StreamMessageBuildInfo:
		m.building = true
		m.tokensByPath[msg.BuildInfo.Path] += msg.BuildInfo.NumTokens
		if msg.BuildInfo.Finished {
			m.finishedByPath[msg.BuildInfo.Path] = true
		}

	case shared.StreamMessageDescribing:
		m.processing = true
		return m, m.spinner.Tick

	case shared.StreamMessageError:

	case shared.StreamMessageFinished:

	case shared.StreamMessageAborted:
	}

	return m, nil
}
