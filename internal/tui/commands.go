// internal/tui/commands.go
package tui

import (
	"time"

	"github.com/sky-xo/june/internal/agent"
	"github.com/sky-xo/june/internal/claude"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages for the TUI
type (
	tickMsg       time.Time
	channelsMsg   []agent.Channel
	transcriptMsg struct {
		agentID string
		entries []claude.Entry
	}
	errMsg error
)

// tickCmd returns a command that ticks every second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// scanChannelsCmd scans for channels and their agents.
func scanChannelsCmd(claudeProjectsDir, basePath, repoName string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Pass actual db connection for Codex agent integration
		channels, err := claude.ScanChannels(claudeProjectsDir, basePath, repoName, nil)
		if err != nil {
			return errMsg(err)
		}
		return channelsMsg(channels)
	}
}

// loadTranscriptCmd loads a transcript from a file.
func loadTranscriptCmd(a agent.Agent) tea.Cmd {
	return func() tea.Msg {
		entries, err := claude.ParseTranscript(a.TranscriptPath)
		if err != nil {
			return errMsg(err)
		}
		return transcriptMsg{
			agentID: a.ID,
			entries: entries,
		}
	}
}
