package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"geminielf/internal/ai"
	"geminielf/internal/git"

	"github.com/charmbracelet/bubbles/progress"
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
	progress      progress.Model
	width         int
	height        int
	progressValue float64
}

type msgCommitGenerated struct {
	message string
	err     error
}

type msgCommitDone struct {
	err error
}

type msgProgressUpdate struct {
	value float64
}


var (
	// グラデーションカラーパレット
	primaryGradient = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.AdaptiveColor{
				Light: "#667eea",
				Dark:  "#764ba2",
			})


	// 確認ダイアログスタイル
	confirmStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#667eea")).
			Padding(3, 4).
			Margin(1, 2).
			Width(70).
			Align(lipgloss.Center).
			Background(lipgloss.AdaptiveColor{
				Light: "#f8f9fa",
				Dark:  "#0d1117",
			})

	// コミットメッセージスタイル
	commitMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E6EDF3")).
			Background(lipgloss.Color("#21262D")).
			Padding(1, 2).
			Margin(1, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#30363D")).
			Italic(true)

	// アクションボタンスタイル
	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#28a745")).
			Padding(0, 2).
			Margin(0, 1).
			Bold(true).
			Border(lipgloss.RoundedBorder())

	cancelButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#dc3545")).
			Padding(0, 2).
			Margin(0, 1).
			Bold(true).
			Border(lipgloss.RoundedBorder())

	// 成功メッセージスタイル
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#28a745")).
			Padding(2, 3).
			Margin(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#28a745")).
			Bold(true)

	// エラーメッセージスタイル
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#dc3545")).
			Padding(2, 3).
			Margin(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#dc3545")).
			Bold(true)

	// ローディングスタイル
	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#667eea")).
			Bold(true).
			Margin(1, 0)

	// ヘルプテキストスタイル
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B949E")).
			Align(lipgloss.Center).
			MarginTop(1)

)

func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#667eea"))
	
	// プログレスバーの設定
	prog := progress.New(progress.WithDefaultGradient())
	prog.Full = '█'
	prog.Empty = '░'
	prog.Width = 60
	
	return &model{
		aiClient:      aiClient,
		diff:          diff,
		state:         stateLoading,
		spinner:       s,
		progress:      prog,
		width:         80,
		height:        24,
		progressValue: 0.0,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.generateCommitMessage(), m.simulateProgress())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(40, m.width-10)
		
	case tea.KeyMsg:
		switch m.state {
		case stateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = stateCommitting
				m.progressValue = 0.0 // コミット開始時はリセット
				return m, tea.Batch(m.spinner.Tick, m.commitChanges(), m.simulateCommitProgress())
			case "n", "N", "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateSuccess, stateError:
			return m, tea.Quit
		}

	case msgProgressUpdate:
		m.progressValue = msg.value
		// 処理中なら継続的にプログレス更新
		if m.state == stateLoading {
			return m, m.simulateProgress()
		} else if m.state == stateCommitting {
			return m, m.simulateCommitProgress()
		}
		
	case msgCommitGenerated:
		m.progressValue = 1.0 // 完了時は100%
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
			return m, tea.Quit
		} else {
			m.state = stateSuccess
			return m, tea.Quit
		}
		
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
		loadingText := loadingStyle.Render(fmt.Sprintf("%s Generating commit message...", m.spinner.View()))
		progressBar := m.progress.ViewAs(m.progressValue)
		
		// プログレスバーコンテナ
		progressContainer := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#667eea")).
			Padding(2, 3).
			Margin(1, 2).
			Width(70).
			Align(lipgloss.Center).
			Background(lipgloss.AdaptiveColor{
				Light: "#f8f9fa",
				Dark:  "#0d1117",
			})
		
		progressContent := lipgloss.JoinVertical(lipgloss.Center,
			"🚀 geminielf",
			loadingText,
			"",
			"🧠 AI is analyzing your changes...",
			"",
			progressBar,
			fmt.Sprintf("%.0f%%", m.progressValue*100),
		)
		
		progressBox := progressContainer.Render(progressContent)
		return lipgloss.JoinVertical(lipgloss.Center, progressBox)

	case stateConfirm:
		messageBox := commitMessageStyle.Render(m.commitMessage)
		buttons := lipgloss.JoinHorizontal(lipgloss.Center,
			buttonStyle.Render("✓ Yes (y)"),
			cancelButtonStyle.Render("✗ No (n)"),
		)
		helpText := helpStyle.Render("Press 'y' to commit or 'n' to cancel")
		
		content := lipgloss.JoinVertical(lipgloss.Center,
			"🚀 geminielf",
			"",
			"📝 Generated Commit Message:",
			messageBox,
			"",
			"🤔 Commit with this message?",
			buttons,
			helpText,
		)
		
		confirmBox := confirmStyle.Render(content)
		return lipgloss.JoinVertical(lipgloss.Center, confirmBox)

	case stateCommitting:
		committingText := loadingStyle.Render(fmt.Sprintf("%s Committing changes...", m.spinner.View()))
		progressBar := m.progress.ViewAs(m.progressValue)
		
		// コミット用プログレスバーコンテナ
		commitContainer := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#28a745")).
			Padding(2, 3).
			Margin(1, 2).
			Width(70).
			Align(lipgloss.Center).
			Background(lipgloss.AdaptiveColor{
				Light: "#f8f9fa",
				Dark:  "#0d1117",
			})
		
		commitContent := lipgloss.JoinVertical(lipgloss.Center,
			"🚀 geminielf",
			committingText,
			"",
			"💾 Applying changes to repository...",
			"",
			progressBar,
			fmt.Sprintf("%.0f%%", m.progressValue*100),
		)
		
		commitBox := commitContainer.Render(commitContent)
		return lipgloss.JoinVertical(lipgloss.Center, commitBox)

	case stateSuccess:
		// 成功アイコンとタイトル
		successTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#28a745")).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(1).
			Render("🎉 Success!")
		
		// 成功メッセージコンテナ
		successContainer := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#28a745")).
			Padding(3, 4).
			Margin(1, 2).
			Width(80).
			Align(lipgloss.Center).
			Background(lipgloss.AdaptiveColor{
				Light: "#f8f9fa",
				Dark:  "#0d1117",
			})
		
		// コミットしたメッセージを表示
		commitedMessageBox := commitMessageStyle.Render(m.commitMessage)
		
		successContent := lipgloss.JoinVertical(lipgloss.Center,
			"🚀 geminielf",
			"",
			successTitle,
			"",
			"✨ Your changes have been committed successfully!",
			"",
			"📝 Committed with message:",
			commitedMessageBox,
			"",
			"🚀 The AI-generated message has been applied.",
		)
		
		successBox := successContainer.Render(successContent)
		return lipgloss.JoinVertical(lipgloss.Center, successBox)

	case stateError:
		// エラーアイコンとタイトル
		errorTitle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#dc3545")).
			Bold(true).
			Align(lipgloss.Center).
			MarginBottom(1).
			Render("❌ Error Occurred")
		
		// エラーメッセージコンテナ
		errorContainer := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#dc3545")).
			Padding(3, 4).
			Margin(1, 2).
			Width(70).
			Align(lipgloss.Center).
			Background(lipgloss.AdaptiveColor{
				Light: "#f8f9fa",
				Dark:  "#0d1117",
			})
		
		errorContent := lipgloss.JoinVertical(lipgloss.Center,
			"🚀 geminielf",
			"",
			errorTitle,
			"",
			fmt.Sprintf("🔍 Details: %v", m.err),
			"",
			"💡 Please check your configuration and try again.",
			"🔧 Make sure Git is properly configured.",
		)
		
		errorBox := errorContainer.Render(errorContent)
		return lipgloss.JoinVertical(lipgloss.Center, errorBox)
	}

	return ""
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


// AI処理中のプログレス更新をシミュレート
func (m *model) simulateProgress() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		if m.state == stateLoading {
			// プログレスを徐々に増加（最大90%まで）
			newValue := m.progressValue + 0.02
			if newValue > 0.9 {
				newValue = 0.9
			}
			return msgProgressUpdate{value: newValue}
		}
		return nil
	})
}

// コミット処理中のプログレス更新をシミュレート  
func (m *model) simulateCommitProgress() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		if m.state == stateCommitting {
			// コミット処理は高速なのでより速くプログレス
			newValue := m.progressValue + 0.1
			if newValue > 1.0 {
				newValue = 1.0
			}
			return msgProgressUpdate{value: newValue}
		}
		return nil
	})
}


func (m *model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

// ヘルパー関数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}