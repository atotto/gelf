# 🚀 gelf

gelf is a Go-based CLI tool that automatically generates Git commit messages and provides AI-powered code reviews using Vertex AI (Gemini). It analyzes git changes and provides intelligent feedback through a modern, interactive TUI interface built with Bubble Tea.

## ✨ Features

- 🤖 **AI-Powered**: Intelligent commit message generation using Vertex AI (Gemini)
- 🔍 **Code Review**: AI-powered code review with streaming real-time feedback
- 🎨 **Clean TUI**: Simple and intuitive user interface built with Bubble Tea  
- ⚡ **Fast Processing**: Real-time progress indicators and streaming responses
- 🛡️ **Safe Operations**: Only operates on staged changes for secure workflow
- 🌐 **Cross-Platform**: Works seamlessly across different operating systems

## 🛠️ Installation

### Prerequisites

- Go 1.24.3 or higher
- Google Cloud account with Vertex AI API enabled
- Git (required for commit operations)

### Build from Source

```bash
git clone https://github.com/EkeMinusYou/gelf.git
cd gelf
go build
```

### Install Binary

```bash
go install github.com/EkeMinusYou/gelf@latest
```

### Install via Homebrew

```bash
brew tap ekeminusyou/gelf
brew install ekeminusyou/gelf/gelf
```

## ⚙️ Setup

### 1. Configuration Options

gelf supports both configuration files and environment variables. Configuration files provide a more organized approach for managing settings.

#### Configuration File (Recommended)

Create a `gelf.yml` file in one of the following locations (in order of priority):

1. `./gelf.yml` - Project-specific configuration
2. `$XDG_CONFIG_HOME/gelf/gelf.yml` - XDG config directory
3. `~/.config/gelf/gelf.yml` - Default XDG config location
4. `~/.gelf.yml` - Legacy home directory location

```yaml
vertex_ai:
  project_id: "your-gcp-project-id"
  location: "us-central1"  # optional, default: us-central1

gelf:
  flash_model: "gemini-2.5-flash-preview-05-20"  # optional
  pro_model: "gemini-2.5-pro-preview-05-06"       # optional
```

#### Environment Variables (Alternative)

You can also configure using environment variables:

```bash
# Path to your service account key file
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"

# Google Cloud project ID
export VERTEXAI_PROJECT="your-project-id"

# Vertex AI location (optional, default: us-central1)
export VERTEXAI_LOCATION="us-central1"
```

**Note**: Model configuration (flash_model and pro_model) can only be configured via configuration file, not environment variables.

### 2. Google Cloud Authentication

1. Create a service account in Google Cloud Console
2. Grant the "Vertex AI User" role
3. Download the JSON key file
4. Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the file path

## 🚀 Usage

### Commit Message Generation

1. Stage your changes:
```bash
git add .
```

2. Generate and commit with AI:
```bash
gelf commit
```

3. Interactive TUI operations:
   - Review the AI-generated commit message
   - Press `y` to approve or `n` to cancel
   - Press `e` to edit the commit message
   - Press `q` or `Ctrl+C` to cancel during generation
   - The commit will be executed automatically upon approval
   - Success message displays after TUI exits

### Code Review

1. Review unstaged changes:
```bash
gelf review
```

2. Review staged changes:
```bash
gelf review --staged
```

The review feature provides:
   - Real-time streaming AI analysis
   - Comprehensive code review feedback
   - Security vulnerability detection
   - Performance and maintainability suggestions
   - No interactive prompts - displays results directly

### Command Options

```bash
# Show help
gelf --help

# Show commit command help
gelf commit --help

# Show review command help
gelf review --help

# Generate commit message with TUI interface (default behavior)
gelf commit

# Generate commit message only with diff display (for debugging)
gelf commit --dry-run

# Generate commit message only without diff (for external tool integration)
gelf commit --dry-run --quiet

# Use specific model temporarily
gelf commit --model gemini-2.0-flash-exp

# Review unstaged changes (default)
gelf review

# Review staged changes
gelf review --staged

# Use specific model for review
gelf review --model gemini-2.0-flash-exp
```

## 🔧 Technical Specifications

### Architecture

- **Commit Target**: Staged changes only (`git diff --staged`)
- **Review Target**: Both staged (`git diff --staged`) and unstaged (`git diff`) changes
- **AI Provider**: Vertex AI (Gemini models)
- **Default Flash Model**: gemini-2.5-flash-preview-05-20
- **Default Pro Model**: gemini-2.5-pro-preview-05-06
- **UI Framework**: Bubble Tea (TUI)
- **CLI Framework**: Cobra
- **Streaming**: Real-time AI response streaming for code reviews

### Project Structure

```
cmd/
├── root.go          # Root command definition
├── commit.go        # Commit command implementation
└── review.go        # Review command implementation
internal/
├── git/
│   └── diff.go      # Git operations (staged and unstaged diffs)
├── ai/
│   └── vertex.go    # Vertex AI integration (commit messages and code review)
├── ui/
│   └── tui.go       # Bubble Tea TUI implementation (commit and review)
└── config/
    └── config.go    # Configuration management (API keys etc)
main.go             # Application entry point
```

## 🎨 TUI Interface

### Commit Workflow

#### Loading Screen
```
┌─────────────────────────────────────────┐
│ ⠙ Generating commit message...         │
└─────────────────────────────────────────┘
```

#### Confirmation Screen
```
┌────────────────────────────────────────────────┐
│                                                │
│  📝 Generated Commit Message:                 │
│                                                │
│  ┌──────────────────────────────────────────┐ │
│  │ feat: add user authentication system    │ │
│  │ with JWT support                         │ │
│  └──────────────────────────────────────────┘ │
│                                                │
│  Commit this message? (y)es / (n)o / (e)dit   │
│                                                │
└────────────────────────────────────────────────┘
```

#### Committing Screen
```
┌─────────────────────────────────────────┐
│ ⠙ Committing changes...                │
└─────────────────────────────────────────┘
```

#### Success Screen
```
┌─────────────────────────────────────────────────┐
│ ✓ Committed: feat: add user authentication     │
│   system with JWT support                       │
└─────────────────────────────────────────────────┘
```

### Review Workflow

#### Loading Screen
```
┌─────────────────────────────────────────┐
│ ⠙ Analyzing code for review...         │
└─────────────────────────────────────────┘
```

#### Streaming Review Display
The review results are displayed in real-time as streaming text without frames, providing immediate feedback as the AI analyzes the code changes.

### Error Screens
```
┌─────────────────────────────────────────┐
│ ✗ Error: No staged changes found       │
└─────────────────────────────────────────┘
```

The interface features:
- **Cyan colored** loading messages with animated spinners (using Points spinner style)
- **Rounded border frames** for loading and confirmation states
- **Dark background colored boxes** for commit messages with italic text
- **Streaming text output** for code reviews without UI frames
- **Color-coded states**: Cyan for loading/generating, Blue for confirmation, Green for success, Red for errors
- **Icon prefixes**: 📝 for messages, 🔍 for reviews, ✓ for success, ✗ for errors
- **Simple, clean design** with proper spacing and padding

## ⚙️ Configuration Reference

### Configuration Priority

Settings are applied in the following order (highest to lowest priority):

1. **Environment variables** (for Vertex AI settings only)
2. **Configuration file** (`gelf.yml`)
3. **Default values**

### Configuration File Options

```yaml
vertex_ai:
  project_id: string     # Google Cloud project ID
  location: string       # Vertex AI location (default: us-central1)

gelf:
  flash_model: string    # Gemini Flash model to use (default: gemini-2.5-flash-preview-05-20)
  pro_model: string      # Gemini Pro model to use (default: gemini-2.5-pro-preview-05-06)
```

### Environment Variables

| Variable | Description | Default Value | Required |
|----------|-------------|---------------|----------|
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account key file | - | ✅ |
| `VERTEXAI_PROJECT` or `GOOGLE_CLOUD_PROJECT` | Google Cloud project ID | - | ✅ |
| `VERTEXAI_LOCATION` | Vertex AI location | `us-central1` | ❌ |

**Note**: Model configuration (flash_model and pro_model) is only available through configuration files.

## 🔨 Development

### Development Environment Setup

```bash
# Install dependencies
go mod download

# Build the project
go build

# Run tests
go test ./...

# Tidy dependencies
go mod tidy
```

### Available Commands

```bash
go build                     # Build the project
go test ./...                # Run tests
go mod tidy                  # Tidy dependencies
go run main.go commit        # Run commit command in development
go run main.go commit --dry-run  # Run message generation only
go run main.go review        # Run review command in development
go run main.go review --staged  # Run review for staged changes
```

## 📦 Dependencies

### Main Dependencies
- [`google.golang.org/genai`](https://pkg.go.dev/google.golang.org/genai) - Official Gemini Go client
- [`github.com/charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) - TUI framework
- [`github.com/charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) - Styling and layout
- [`github.com/charmbracelet/bubbles`](https://github.com/charmbracelet/bubbles) - TUI components (spinner)
- [`github.com/spf13/cobra`](https://github.com/spf13/cobra) - CLI framework
- [`gopkg.in/yaml.v3`](https://gopkg.in/yaml.v3) - YAML configuration file support

## 🤝 Contributing

Pull requests and issues are welcome!

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a pull request

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - For enabling beautiful TUI experiences
- [Vertex AI](https://cloud.google.com/vertex-ai) - For providing powerful AI capabilities
- [Cobra](https://github.com/spf13/cobra) - For excellent CLI experience

---

**Made with ❤️ by [EkeMinusYou](https://github.com/EkeMinusYou)**