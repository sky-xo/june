// internal/tui/model.go
package tui

import (
	"fmt"
	"strings"

	"june/internal/claude"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 20

var (
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")) // cyan
)

// Model is the TUI state.
type Model struct {
	projectDir  string                    // Claude project directory we're watching
	agents      []claude.Agent            // List of agents
	transcripts map[string][]claude.Entry // Agent ID -> transcript entries

	selectedIdx int            // Currently selected agent index
	width       int
	height      int
	viewport    viewport.Model
	err         error
}

// NewModel creates a new TUI model.
func NewModel(projectDir string) Model {
	return Model{
		projectDir:  projectDir,
		agents:      []claude.Agent{},
		transcripts: make(map[string][]claude.Entry),
		viewport:    viewport.New(0, 0),
	}
}

// SelectedAgent returns the currently selected agent, or nil if none.
func (m Model) SelectedAgent() *claude.Agent {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.agents) {
		return nil
	}
	return &m.agents[m.selectedIdx]
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		scanAgentsCmd(m.projectDir),
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				if agent := m.SelectedAgent(); agent != nil {
					cmds = append(cmds, loadTranscriptCmd(*agent))
				}
			}
		case "down", "j":
			if m.selectedIdx < len(m.agents)-1 {
				m.selectedIdx++
				if agent := m.SelectedAgent(); agent != nil {
					cmds = append(cmds, loadTranscriptCmd(*agent))
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - sidebarWidth - 4
		m.viewport.Height = msg.Height - 4

	case tickMsg:
		cmds = append(cmds, tickCmd(), scanAgentsCmd(m.projectDir))

	case agentsMsg:
		m.agents = msg
		// Load transcript for selected agent
		if agent := m.SelectedAgent(); agent != nil {
			cmds = append(cmds, loadTranscriptCmd(*agent))
		}

	case transcriptMsg:
		m.transcripts[msg.agentID] = msg.entries
		m.updateViewport()

	case errMsg:
		m.err = msg
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateViewport() {
	agent := m.SelectedAgent()
	if agent == nil {
		m.viewport.SetContent("")
		return
	}
	entries := m.transcripts[agent.ID]
	m.viewport.SetContent(formatTranscript(entries))
}

// View renders the UI.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	// Left panel: agent list
	sidebar := m.renderSidebar()

	// Right panel: transcript
	content := m.viewport.View()

	// Combine
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", content)
}

func (m Model) renderSidebar() string {
	var lines []string
	lines = append(lines, "Subagents")
	lines = append(lines, strings.Repeat("-", sidebarWidth-2))

	for i, agent := range m.agents {
		var indicator string
		if agent.IsActive() {
			indicator = activeStyle.Render("●")
		} else {
			indicator = doneStyle.Render("✓")
		}

		name := agent.ID
		if len(name) > sidebarWidth-4 {
			name = name[:sidebarWidth-4]
		}

		line := fmt.Sprintf(" %s %s", indicator, name)
		if i == m.selectedIdx {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	// Pad to full height
	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func formatTranscript(entries []claude.Entry) string {
	var lines []string
	for _, e := range entries {
		switch e.Type {
		case "user":
			content := e.TextContent()
			if content != "" {
				lines = append(lines, fmt.Sprintf("> %s", content))
				lines = append(lines, "")
			}
		case "assistant":
			if tool := e.ToolName(); tool != "" {
				lines = append(lines, fmt.Sprintf("  [%s]", tool))
			} else if text := e.TextContent(); text != "" {
				lines = append(lines, text)
				lines = append(lines, "")
			}
		}
	}
	return strings.Join(lines, "\n")
}
