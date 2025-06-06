package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"geminielf/internal/ai"
	"geminielf/internal/git"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateLoading state = iota
	stateConfirm
	stateCommitting
	stateSuccess
	stateError
)

type model struct {
	aiClient      *ai.VertexAIClient
	diff          string
	commitMessage string
	err           error
	state         state
	spinner       spinner.Model
	progress      float64
}

type msgCommitGenerated struct {
	message string
	err     error
}

type msgCommitDone struct {
	err error
}

type msgProgressUpdate struct {
	progress float64
}




var (
	// エレガントなスタイル
	titleStyle = lipgloss.NewStyle().Bold(true)
	
	messageStyle = lipgloss.NewStyle().
		Padding(1, 0).
		Italic(true)

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))

	// 生成中の洗練されたスタイル
	generatingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	progressBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4"))

	loadingFrameStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Margin(1, 0)
)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	
	return &model{
		aiClient: aiClient,
		diff:     diff,
		state:    stateLoading,
		spinner:  s,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.generateCommitMessage(), m.updateProgress())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateLoading:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = stateCommitting
				m.progress = 0.0
				return m, tea.Batch(m.spinner.Tick, m.commitChanges(), m.updateProgress())
			case "n", "N", "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateSuccess, stateError:
			return m, tea.Quit
		}

	case msgProgressUpdate:
		m.progress = msg.progress
		if m.state == stateLoading || m.state == stateCommitting {
			return m, m.updateProgress()
		}

	case msgCommitGenerated:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
		} else {
			m.commitMessage = msg.message
			m.state = stateConfirm
		}

	case msgCommitDone:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
		} else {
			m.state = stateSuccess
		}
		return m, tea.Quit
	}

	// Update spinner
	if m.state == stateLoading || m.state == stateCommitting {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) View() string {
	switch m.state {
	case stateLoading:
		progressBar := m.renderProgressBar(m.progress)
		content := fmt.Sprintf("%s %s\n\n%s",
			m.spinner.View(),
			generatingStyle.Render("Generating commit message..."),
			progressBar)
		return loadingFrameStyle.Render(content)

	case stateConfirm:
		return fmt.Sprintf("%s\nCommit this message? %s",
			messageStyle.Render(m.commitMessage),
			promptStyle.Render("(y)es / (n)o"))

	case stateCommitting:
		progressBar := m.renderProgressBar(m.progress)
		content := fmt.Sprintf("%s %s\n\n%s",
			m.spinner.View(),
			generatingStyle.Render("Committing changes..."),
			progressBar)
		return loadingFrameStyle.Render(content)

	case stateSuccess:
		return ""

	case stateError:
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return ""
}

func (m *model) renderProgressBar(progress float64) string {
	width := 30
	filled := int(progress * float64(width))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	percentage := fmt.Sprintf("%.0f%%", progress*100)
	
	return fmt.Sprintf("%s %s",
		progressBarStyle.Render(bar),
		progressBarStyle.Render(percentage))
}

func (m *model) generateCommitMessage() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		message, err := m.aiClient.GenerateCommitMessage(ctx, m.diff)
		return msgCommitGenerated{
			message: strings.TrimSpace(message),
			err:     err,
		}
	})
}

func (m *model) commitChanges() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		err := git.CommitChanges(m.commitMessage)
		return msgCommitDone{err: err}
	})
}

func (m *model) updateProgress() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		if m.state == stateLoading {
			// Progress gradually increases to 90% max during loading
			newProgress := m.progress + 0.02
			if newProgress > 0.9 {
				newProgress = 0.9
			}
			return msgProgressUpdate{progress: newProgress}
		} else if m.state == stateCommitting {
			// Faster progress during commit
			newProgress := m.progress + 0.1
			if newProgress > 1.0 {
				newProgress = 1.0
			}
			return msgProgressUpdate{progress: newProgress}
		}
		return nil
	})
}





func (m *model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	
	// Print success message after TUI exits so it remains visible
	if m.state == stateSuccess {
		fmt.Print(successStyle.Render(fmt.Sprintf("✓ Committed: %s", m.commitMessage)) + "\n")
	}
	
	return err
}

