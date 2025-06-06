package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gelf/internal/ai"
	"gelf/internal/git"

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
	startTime     time.Time
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
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))
	
	messageStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Background(lipgloss.Color("235")).
		Margin(0).
		Italic(true)

	commitMessageHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).
		Bold(true).
		Margin(1, 0, 0, 0)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("1")).
		Padding(0, 1).
		Margin(1, 0)

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

	confirmFrameStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("4")).
		Padding(1, 2).
		Margin(1, 0)
)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	
	return &model{
		aiClient:  aiClient,
		diff:      diff,
		state:     stateLoading,
		spinner:   s,
		startTime: time.Now(),
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
				m.startTime = time.Now()
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
			m.progress = 1.0 // Complete progress when AI response received
		}

	case msgCommitDone:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
		} else {
			m.state = stateSuccess
			m.progress = 1.0 // Complete progress when commit done
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
		header := commitMessageHeaderStyle.Render("📝 Generated Commit Message:")
		message := messageStyle.Render(m.commitMessage)
		prompt := promptStyle.Render("Commit this message? (y)es / (n)o")
		
		content := fmt.Sprintf("%s\n\n%s\n%s", header, message, prompt)
		return confirmFrameStyle.Render(content)

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
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err))
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
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		if m.state == stateLoading {
			// Progress based on elapsed time (estimate: AI response takes ~5-10 seconds)
			elapsed := time.Since(m.startTime).Seconds()
			estimatedDuration := 8.0 // 8 seconds estimate
			newProgress := elapsed / estimatedDuration
			if newProgress > 0.95 {
				newProgress = 0.95 // Cap at 95% until actual completion
			}
			return msgProgressUpdate{progress: newProgress}
		} else if m.state == stateCommitting {
			// Progress based on elapsed time (estimate: commit takes ~1-2 seconds)
			elapsed := time.Since(m.startTime).Seconds()
			estimatedDuration := 1.5 // 1.5 seconds estimate
			newProgress := elapsed / estimatedDuration
			if newProgress > 0.95 {
				newProgress = 0.95 // Cap at 95% until actual completion
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
		successFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2")).
			Padding(0, 1).
			Margin(1, 0)
		
		successMessage := successStyle.Render(fmt.Sprintf("✓ Committed: %s", m.commitMessage))
		fmt.Print(successFrame.Render(successMessage) + "\n")
	}
	
	return err
}

