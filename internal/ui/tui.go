package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/EkeMinusYou/gelf/internal/ai"
	"github.com/EkeMinusYou/gelf/internal/git"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateLoading state = iota
	stateConfirm
	stateEditing
	stateCommitting
	stateSuccess
	stateError
)

type model struct {
	aiClient        *ai.VertexAIClient
	diff            string
	diffSummary     git.DiffSummary
	commitMessage   string
	originalMessage string
	err             error
	state           state
	spinner         spinner.Model
	textInput       textinput.Model
}

type msgCommitGenerated struct {
	message string
	err     error
}

type msgCommitDone struct {
	err error
}






func NewTUI(aiClient *ai.VertexAIClient, diff string) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = loadingStyle
	
	ti := textinput.New()
	ti.Placeholder = "Enter your commit message..."
	ti.CharLimit = 200
	ti.Width = 60
	
	diffSummary := git.ParseDiffSummary(diff)
	
	return &model{
		aiClient:    aiClient,
		diff:        diff,
		diffSummary: diffSummary,
		state:       stateLoading,
		spinner:     s,
		textInput:   ti,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.generateCommitMessage())
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
				return m, tea.Batch(m.spinner.Tick, m.commitChanges())
			case "e", "E":
				m.originalMessage = m.commitMessage
				m.textInput.SetValue(m.commitMessage)
				m.textInput.Focus()
				m.state = stateEditing
				return m, textinput.Blink
			case "n", "N", "q", "ctrl+c":
				return m, tea.Quit
			}
		case stateEditing:
			switch msg.String() {
			case "enter":
				m.commitMessage = strings.TrimSpace(m.textInput.Value())
				if m.commitMessage == "" {
					m.commitMessage = m.originalMessage
				}
				m.textInput.Blur()
				m.state = stateConfirm
			case "esc":
				m.commitMessage = m.originalMessage
				m.textInput.Blur()
				m.state = stateConfirm
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		case stateSuccess, stateError:
			return m, tea.Quit
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
		loadingText := fmt.Sprintf("%s %s", 
			m.spinner.View(), 
			loadingStyle.Render("Generating commit message..."))
		
		diffSummary := m.formatDiffSummary()
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s", diffSummary, loadingText)
		}
		return loadingText

	case stateConfirm:
		diffSummary := m.formatDiffSummary()
		header := titleStyle.Render("📝 Generated Commit Message:")
		message := messageStyle.Render(m.commitMessage)
		prompt := promptStyle.Render("Commit this message? (y)es / (e)dit / (n)o")
		
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", diffSummary, header, message, prompt)
		}
		return fmt.Sprintf("%s\n\n%s\n\n%s", header, message, prompt)

	case stateEditing:
		diffSummary := m.formatDiffSummary()
		header := titleStyle.Render("✏️  Edit Commit Message:")
		inputView := m.textInput.View()
		prompt := editPromptStyle.Render("Press Enter to confirm, Esc to cancel")
		
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s", diffSummary, header, inputView, prompt)
		}
		return fmt.Sprintf("%s\n\n%s\n\n%s", header, inputView, prompt)

	case stateCommitting:
		return fmt.Sprintf("%s %s", 
			m.spinner.View(), 
			loadingStyle.Render("Committing changes..."))

	case stateSuccess:
		return ""

	case stateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err))
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

func (m *model) formatDiffSummary() string {
	if len(m.diffSummary.Files) == 0 {
		return ""
	}
	
	var parts []string
	parts = append(parts, diffStyle.Render("📄 Changed Files:"))
	
	for _, file := range m.diffSummary.Files {
		fileName := fileStyle.Render(file.Name)
		
		var changes []string
		if file.AddedLines > 0 {
			changes = append(changes, addedStyle.Render(fmt.Sprintf("+%d", file.AddedLines)))
		}
		if file.DeletedLines > 0 {
			changes = append(changes, deletedStyle.Render(fmt.Sprintf("-%d", file.DeletedLines)))
		}
		
		if len(changes) > 0 {
			parts = append(parts, fmt.Sprintf(" • %s (%s)", fileName, strings.Join(changes, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf(" • %s", fileName))
		}
	}
	
	return strings.Join(parts, "\n")
}






func (m *model) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	
	// Print success message after TUI exits so it remains visible
	if m.state == stateSuccess {
		header := successStyle.Render("✓ Commit successful")
		message := messageStyle.Render(m.commitMessage)
		
		fmt.Printf("%s\n%s\n", header, message)
	}
	
	return err
}

// Review TUI model
type reviewModel struct {
	aiClient *ai.VertexAIClient
	diff     string
	diffSummary git.DiffSummary
	review   string
	structuredReview *ai.StructuredReview
	err      error
	state    reviewState
	spinner  spinner.Model
	sub      chan msgReviewChunk
	noStyle  bool
	renderer *glamour.TermRenderer
	useStructured bool
}

type reviewState int

const (
	reviewStateLoading reviewState = iota
	reviewStateStreaming
	reviewStateDisplay
	reviewStateError
)

type msgReviewGenerated struct {
	review string
	err    error
}

type msgStructuredReviewGenerated struct {
	review *ai.StructuredReview
	err    error
}

type msgReviewChunk struct {
	chunk string
}

type msgReviewComplete struct {
	err error
}

// リッチなカラースタイル（枠なし）
var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")) // シアン

	messageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).   // ブライトホワイト
		Bold(true).
		Italic(true)

	promptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4")).    // ブルー
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).    // グリーン
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).    // レッド
		Bold(true)

	loadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).    // シアン
		Bold(true)

	editPromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).    // イエロー
		Bold(true).
		Margin(1, 0, 0, 0)
	
	diffStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).    // ライトグレー
		Margin(1, 0, 0, 0)
	
	fileStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).    // マゼンタ
		Bold(true)
	
	addedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))     // グリーン
	
	deletedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))     // レッド

)

func NewReviewTUI(aiClient *ai.VertexAIClient, diff string, noStyle bool) *reviewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = loadingStyle
	
	var renderer *glamour.TermRenderer
	if !noStyle {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(80),
		)
		if err == nil {
			renderer = r
		}
	}
	
	diffSummary := git.ParseDiffSummary(diff)
	
	return &reviewModel{
		aiClient:      aiClient,
		diff:          diff,
		diffSummary:   diffSummary,
		state:         reviewStateLoading,
		spinner:       s,
		sub:           make(chan msgReviewChunk, 100),
		noStyle:       noStyle,
		renderer:      renderer,
		useStructured: true, // Use structured review by default
	}
}

// SetLegacyMode enables legacy streaming review mode
func (m *reviewModel) SetLegacyMode(legacy bool) {
	m.useStructured = !legacy
}

func (m *reviewModel) Init() tea.Cmd {
	if m.useStructured {
		return tea.Batch(m.spinner.Tick, m.generateStructuredReview())
	}
	return tea.Batch(m.spinner.Tick, m.generateReviewStreaming(), m.waitForActivity(m.sub))
}

func (m *reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case reviewStateLoading, reviewStateStreaming:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		case reviewStateDisplay, reviewStateError:
			return m, tea.Quit
		}

	case msgReviewChunk:
		if m.state == reviewStateLoading {
			m.state = reviewStateStreaming
		}
		m.review += msg.chunk
		return m, m.waitForActivity(m.sub)

	case msgReviewComplete:
		if msg.err != nil {
			m.err = msg.err
			m.state = reviewStateError
		} else {
			m.state = reviewStateDisplay
		}
		return m, tea.Quit

	case msgReviewGenerated:
		if msg.err != nil {
			m.err = msg.err
			m.state = reviewStateError
		} else {
			m.review = msg.review
			m.state = reviewStateDisplay
		}
		return m, tea.Quit

	case msgStructuredReviewGenerated:
		if msg.err != nil {
			m.err = msg.err
			m.state = reviewStateError
		} else {
			m.structuredReview = msg.review
			m.state = reviewStateDisplay
		}
		return m, tea.Quit
	}

	// Update spinner
	if m.state == reviewStateLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *reviewModel) View() string {
	switch m.state {
	case reviewStateLoading:
		loadingText := fmt.Sprintf("%s %s", 
			m.spinner.View(), 
			loadingStyle.Render("Analyzing code for review..."))
		
		diffSummary := m.formatReviewDiffSummary()
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s", diffSummary, loadingText)
		}
		return loadingText

	case reviewStateStreaming:
		// Display streaming content with styling if available
		var content string
		if m.noStyle || m.renderer == nil {
			content = m.review
		} else {
			// Try to render the current content with glamour
			styled, err := m.renderer.Render(m.review)
			if err != nil {
				// Fallback to plain text if rendering fails
				content = m.review
			} else {
				content = styled
			}
		}
		
		diffSummary := m.formatReviewDiffSummary()
		if diffSummary != "" {
			return fmt.Sprintf("%s\n\n%s", diffSummary, content)
		}
		return content

	case reviewStateDisplay:
		return "" // Review will be printed after TUI exits

	case reviewStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %v", m.err))
	}

	return ""
}


func (m *reviewModel) generateReviewStreaming() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		
		go func() {
			defer close(m.sub)
			
			err := m.aiClient.ReviewCodeStreaming(ctx, m.diff, func(chunk string) {
				select {
				case m.sub <- msgReviewChunk{chunk: chunk}:
				case <-ctx.Done():
					return
				}
			})
			
			if err != nil {
				m.sub <- msgReviewChunk{} // Signal error by closing
			}
		}()
		
		return nil
	})
}

func (m *reviewModel) waitForActivity(sub chan msgReviewChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-sub
		if !ok {
			// Channel closed, streaming completed
			return msgReviewComplete{err: nil}
		}
		return chunk
	}
}

func (m *reviewModel) Run() (string, error) {
	p := tea.NewProgram(m)
	_, err := p.Run()
	
	if m.err != nil {
		return "", m.err
	}
	
	// Print the review after TUI exits
	if m.useStructured && m.structuredReview != nil {
		m.printStructuredReview()
	} else if m.review != "" {
		var content string
		if m.noStyle || m.renderer == nil {
			content = m.review
		} else {
			styled, renderErr := m.renderer.Render(m.review)
			if renderErr != nil {
				content = m.review
			} else {
				content = styled
			}
		}
		
		diffSummary := m.formatReviewDiffSummary()
		if diffSummary != "" {
			fmt.Printf("%s\n\n%s\n", diffSummary, content)
		} else {
			fmt.Print(content + "\n")
		}
	}
	
	return m.review, err
}



// generateStructuredReview generates a structured review
func (m *reviewModel) generateStructuredReview() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		review, err := m.aiClient.ReviewCodeStructured(ctx, m.diff)
		return msgStructuredReviewGenerated{
			review: review,
			err:    err,
		}
	})
}

// printStructuredReview prints the structured review with diff and comments
func (m *reviewModel) printStructuredReview() {
	if m.structuredReview == nil {
		return
	}
	
	// Print overall summary
	fmt.Printf("%s %s\n", 
		titleStyle.Render("📋 Code Review Summary:"), 
		m.structuredReview.Summary)
	fmt.Println()
	
	// Print each file with its diff and comments
	for _, fileReview := range m.structuredReview.FileReviews {
		m.printFileReview(fileReview)
	}
}

// printFileReview prints a single file's review with diff and comments
func (m *reviewModel) printFileReview(fileReview ai.FileReview) {
	// File header
	fileHeader := fmt.Sprintf("📄 %s", fileReview.FileName)
	fmt.Println(fileStyle.Render(fileHeader))
	fmt.Println(strings.Repeat("─", len(fileHeader)))
	
	// Print comments first if any exist
	if len(fileReview.Comments) > 0 {
		fmt.Printf("%s\n", diffStyle.Render("Review Comments:"))
		for _, comment := range fileReview.Comments {
			m.printComment(comment)
		}
		fmt.Println()
		
		// Print relevant diff sections near comments
		if fileReview.DiffText != "" {
			fmt.Printf("%s\n", diffStyle.Render("Relevant Code Changes:"))
			m.printRelevantDiffSections(fileReview.DiffText, fileReview.Comments)
		}
	} else {
		fmt.Printf("%s\n", successStyle.Render("✓ No issues found in this file"))
	}
	
	fmt.Println()
}

// printDiffWithSyntaxHighlight prints diff with basic syntax highlighting, showing only relevant lines
func (m *reviewModel) printDiffWithSyntaxHighlight(diff string) {
	lines := strings.Split(diff, "\n")
	relevantLines := m.extractRelevantDiffLines(lines)
	
	for _, line := range relevantLines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			fmt.Println(addedStyle.Render(line))
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			fmt.Println(deletedStyle.Render(line))
		} else if strings.HasPrefix(line, "@@") {
			fmt.Println(titleStyle.Render(line))
		} else if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index") {
			fmt.Println(diffStyle.Render(line))
		} else {
			fmt.Println(line)
		}
	}
}

// extractRelevantDiffLines extracts only the important lines from a diff
func (m *reviewModel) extractRelevantDiffLines(lines []string) []string {
	var result []string
	var currentHunk []string
	inHunk := false
	hasChanges := false
	
	for i, line := range lines {
		// Header lines (always include)
		if strings.HasPrefix(line, "diff --git") || 
		   strings.HasPrefix(line, "index") ||
		   strings.HasPrefix(line, "+++") ||
		   strings.HasPrefix(line, "---") {
			result = append(result, line)
			continue
		}
		
		// Hunk header
		if strings.HasPrefix(line, "@@") {
			// Process previous hunk if it had changes
			if inHunk && hasChanges {
				result = append(result, currentHunk...)
			}
			
			// Start new hunk
			currentHunk = []string{line}
			inHunk = true
			hasChanges = false
			continue
		}
		
		if inHunk {
			currentHunk = append(currentHunk, line)
			
			// Check if this line is a change (added or removed)
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				hasChanges = true
			}
			
			// If this is the last line or next line starts a new hunk/file
			if i == len(lines)-1 || 
			   (i+1 < len(lines) && (strings.HasPrefix(lines[i+1], "@@") || 
			   	strings.HasPrefix(lines[i+1], "diff --git"))) {
				
				if hasChanges {
					// Only include context lines around changes
					result = append(result, m.filterHunkLines(currentHunk)...)
				}
				inHunk = false
				hasChanges = false
			}
		}
	}
	
	return result
}

// filterHunkLines filters a hunk to show only changed lines with minimal context
func (m *reviewModel) filterHunkLines(hunkLines []string) []string {
	if len(hunkLines) == 0 {
		return hunkLines
	}
	
	// Always include the hunk header
	result := []string{hunkLines[0]}
	
	// Find changed lines (+ or -)
	changedIndices := make(map[int]bool)
	for i := 1; i < len(hunkLines); i++ {
		line := hunkLines[i]
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			changedIndices[i] = true
		}
	}
	
	// If no changes, return just the header
	if len(changedIndices) == 0 {
		return result
	}
	
	// Include changed lines with minimal context (1 line before/after)
	contextLines := 1
	included := make(map[int]bool)
	
	for changeIdx := range changedIndices {
		// Include the changed line
		included[changeIdx] = true
		
		// Include context lines
		for j := max(1, changeIdx-contextLines); j <= min(len(hunkLines)-1, changeIdx+contextLines); j++ {
			included[j] = true
		}
	}
	
	// Add included lines in order
	for i := 1; i < len(hunkLines); i++ {
		if included[i] {
			result = append(result, hunkLines[i])
		}
	}
	
	return result
}

// printRelevantDiffSections prints only diff sections that are relevant to comments
func (m *reviewModel) printRelevantDiffSections(diff string, comments []ai.ReviewComment) {
	lines := strings.Split(diff, "\n")
	
	// If no comments have line numbers, show a minimal diff
	commentLines := make(map[int]bool)
	hasLineNumbers := false
	for _, comment := range comments {
		if comment.LineNo > 0 {
			commentLines[comment.LineNo] = true
			hasLineNumbers = true
		}
	}
	
	if !hasLineNumbers {
		// Show only changed lines if no line numbers in comments
		m.printDiffWithSyntaxHighlight(diff)
		return
	}
	
	// Find relevant hunks based on comment line numbers
	relevantLines := m.extractCommentRelevantLines(lines, commentLines)
	
	for _, line := range relevantLines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			fmt.Println(addedStyle.Render(line))
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			fmt.Println(deletedStyle.Render(line))
		} else if strings.HasPrefix(line, "@@") {
			fmt.Println(titleStyle.Render(line))
		} else if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index") {
			fmt.Println(diffStyle.Render(line))
		} else {
			fmt.Println(line)
		}
	}
}

// extractCommentRelevantLines extracts diff lines that are relevant to comment line numbers
func (m *reviewModel) extractCommentRelevantLines(lines []string, commentLines map[int]bool) []string {
	var result []string
	currentLineNum := 0
	inHunk := false
	contextWindow := 3 // Show 3 lines before/after comment lines
	
	for i, line := range lines {
		// Header lines (always include)
		if strings.HasPrefix(line, "diff --git") || 
		   strings.HasPrefix(line, "index") ||
		   strings.HasPrefix(line, "+++") ||
		   strings.HasPrefix(line, "---") {
			result = append(result, line)
			continue
		}
		
		// Hunk header - parse line numbers
		if strings.HasPrefix(line, "@@") {
			result = append(result, line)
			inHunk = true
			// Extract starting line number from @@ -old_start,old_count +new_start,new_count @@
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				newPart := parts[2] // +new_start,new_count
				if strings.HasPrefix(newPart, "+") {
					newPart = strings.TrimPrefix(newPart, "+")
					if commaIdx := strings.Index(newPart, ","); commaIdx >= 0 {
						newPart = newPart[:commaIdx]
					}
					if num, err := fmt.Sscanf(newPart, "%d", &currentLineNum); err == nil && num == 1 {
						currentLineNum-- // Will be incremented below
					}
				}
			}
			continue
		}
		
		if inHunk {
			// Track line numbers
			if !strings.HasPrefix(line, "-") {
				currentLineNum++
			}
			
			// Check if this line or nearby lines have comments
			isRelevant := false
			for commentLine := range commentLines {
				if abs(currentLineNum-commentLine) <= contextWindow {
					isRelevant = true
					break
				}
			}
			
			if isRelevant {
				result = append(result, line)
			}
			
			// End of hunk
			if i == len(lines)-1 || 
			   (i+1 < len(lines) && (strings.HasPrefix(lines[i+1], "@@") || 
			   	strings.HasPrefix(lines[i+1], "diff --git"))) {
				inHunk = false
			}
		}
	}
	
	return result
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// printComment prints a review comment with appropriate styling
func (m *reviewModel) printComment(comment ai.ReviewComment) {
	var prefix string
	var style lipgloss.Style
	
	switch comment.Type {
	case "must":
		prefix = "🚨 MUST"
		style = errorStyle
	case "want":
		prefix = "💡 WANT"
		style = promptStyle
	case "nits":
		prefix = "✨ NITS"
		style = editPromptStyle
	case "fyi":
		prefix = "ℹ️  FYI"
		style = diffStyle
	case "imo":
		prefix = "💭 IMO"
		style = diffStyle
	default:
		prefix = "💬"
		style = diffStyle
	}
	
	lineInfo := ""
	if comment.LineNo > 0 {
		lineInfo = fmt.Sprintf(" (Line %d)", comment.LineNo)
	}
	
	fmt.Printf("  %s%s: %s\n", 
		style.Render(prefix), 
		lineInfo, 
		comment.Message)
}

func (m *reviewModel) formatReviewDiffSummary() string {
	if len(m.diffSummary.Files) == 0 {
		return ""
	}
	
	var parts []string
	parts = append(parts, diffStyle.Render("📄 Changed Files:"))
	
	for _, file := range m.diffSummary.Files {
		fileName := fileStyle.Render(file.Name)
		
		var changes []string
		if file.AddedLines > 0 {
			changes = append(changes, addedStyle.Render(fmt.Sprintf("+%d", file.AddedLines)))
		}
		if file.DeletedLines > 0 {
			changes = append(changes, deletedStyle.Render(fmt.Sprintf("-%d", file.DeletedLines)))
		}
		
		if len(changes) > 0 {
			parts = append(parts, fmt.Sprintf(" • %s (%s)", fileName, strings.Join(changes, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf(" • %s", fileName))
		}
	}
	
	return strings.Join(parts, "\n")
}

