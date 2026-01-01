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
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))           // green
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // gray
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")) // cyan
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	toolStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// Model is the TUI state.
type Model struct {
	projectDir  string                    // Claude project directory we're watching
	agents      []claude.Agent            // List of agents
	transcripts map[string][]claude.Entry // Agent ID -> transcript entries

	selectedIdx int // Currently selected agent index
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

	// Calculate panel dimensions
	sidebarHeight := m.height
	contentWidth := m.width - sidebarWidth - 1

	// Left panel: agent list with border
	sidebarContent := m.renderSidebarContent()
	sidebar := renderPanelWithTitle("Subagents", sidebarContent, sidebarWidth, sidebarHeight, false)

	// Right panel: transcript with border
	var contentTitle string
	if agent := m.SelectedAgent(); agent != nil {
		contentTitle = agent.ID
	}
	content := renderPanelWithTitle(contentTitle, m.viewport.View(), contentWidth, sidebarHeight, true)

	// Combine horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func (m Model) renderSidebarContent() string {
	var lines []string

	for i, agent := range m.agents {
		var indicator string
		if agent.IsActive() {
			indicator = activeStyle.Render("●")
		} else {
			indicator = doneStyle.Render("✓")
		}

		name := agent.ID
		if len(name) > sidebarWidth-6 {
			name = name[:sidebarWidth-6]
		}

		line := fmt.Sprintf("%s %s", indicator, name)
		if i == m.selectedIdx {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderPanelWithTitle renders a panel with the title embedded in the top border
// like: ╭─ Title ────────╮
func renderPanelWithTitle(title, content string, width, height int, focused bool) string {
	borderColor := lipgloss.Color("8") // dim
	if focused {
		borderColor = lipgloss.Color("6") // cyan
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(borderColor).Bold(true)

	// Border characters (rounded)
	topLeft := "╭"
	topRight := "╮"
	bottomLeft := "╰"
	bottomRight := "╯"
	horizontal := "─"
	vertical := "│"

	contentWidth := width - 2

	// Build top border with embedded title
	var topBorder string
	if title == "" {
		topBorder = borderStyle.Render(topLeft + strings.Repeat(horizontal, contentWidth) + topRight)
	} else {
		titleText := " " + title + " "
		remainingWidth := contentWidth - len(titleText) - 1
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		topBorder = borderStyle.Render(topLeft+horizontal) + titleStyle.Render(titleText) + borderStyle.Render(strings.Repeat(horizontal, remainingWidth)+topRight)
	}

	// Split content into lines and pad/truncate to fit
	contentLines := strings.Split(content, "\n")
	var paddedLines []string
	for i := 0; i < height-2; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		// Truncate if too long
		if len(line) > contentWidth {
			line = line[:contentWidth-1] + "…"
		}
		// Pad to width
		padding := contentWidth - len(line)
		if padding < 0 {
			padding = 0
		}
		paddedLines = append(paddedLines, borderStyle.Render(vertical)+line+strings.Repeat(" ", padding)+borderStyle.Render(vertical))
	}

	// Build bottom border
	bottomBorder := borderStyle.Render(bottomLeft + strings.Repeat(horizontal, contentWidth) + bottomRight)

	// Combine all parts
	result := topBorder + "\n"
	result += strings.Join(paddedLines, "\n") + "\n"
	result += bottomBorder

	return result
}

func formatTranscript(entries []claude.Entry) string {
	var lines []string

	for _, e := range entries {
		switch e.Type {
		case "user":
			if e.IsToolResult() {
				// Skip tool results in display (too verbose)
				continue
			}
			content := e.TextContent()
			if content != "" {
				// Show first 200 chars of prompt
				if len(content) > 200 {
					content = content[:200] + "..."
				}
				lines = append(lines, promptStyle.Render("> "+content))
				lines = append(lines, "")
			}
		case "assistant":
			if tool := e.ToolName(); tool != "" {
				lines = append(lines, toolStyle.Render("  "+tool))
			} else if text := e.TextContent(); text != "" {
				// Show first 500 chars of response
				if len(text) > 500 {
					text = text[:500] + "..."
				}
				lines = append(lines, text)
				lines = append(lines, "")
			}
		}
	}
	return strings.Join(lines, "\n")
}
