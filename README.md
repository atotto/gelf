# 🚀 gelf

gelf is a Go-based CLI tool that automatically generates Git commit messages using Vertex AI (Gemini). It analyzes staged changes and generates appropriate commit messages through a modern, interactive TUI interface built with Bubble Tea.

## ✨ Features

- 🤖 **AI-Powered**: Intelligent commit message generation using Vertex AI (Gemini)
- 🎨 **Clean TUI**: Simple and intuitive user interface built with Bubble Tea  
- ⚡ **Fast Processing**: Real-time progress indicators with percentage display
- 🛡️ **Safe Operations**: Only operates on staged changes for secure workflow
- 🌐 **Cross-Platform**: Works seamlessly across different operating systems

## 🛠️ Installation

### Prerequisites

- Go 1.24.3 or higher
- Google Cloud account with Vertex AI API enabled
- Git (required for commit operations)

### Build from Source

```bash
git clone https://github.com/EkeMinusYou/geminielf.git
cd geminielf
go build
```

### Install Binary

```bash
go install github.com/EkeMinusYou/gelf@latest
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
  default_model: "gemini-2.5-flash-preview-05-20"  # optional
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

**Note**: The default model can only be configured via configuration file, not environment variables.

### 2. Google Cloud Authentication

1. Create a service account in Google Cloud Console
2. Grant the "Vertex AI User" role
3. Download the JSON key file
4. Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the file path

## 🚀 Usage

### Basic Workflow

1. Stage your changes:
```bash
git add .
```

2. Run gelf:
```bash
gelf git commit
```

3. Interactive TUI operations:
   - Review the AI-generated commit message
   - Press `y` to approve or `n` to cancel
   - Press `q` or `Ctrl+C` to cancel during generation
   - The commit will be executed automatically upon approval

### Command Options

```bash
# Show help
gelf --help

# Show git subcommand help
gelf git --help

# Show git commit subcommand help
gelf git commit --help

# Generate commit message only (for external tool integration)
gelf git message

# Show diff with generated message (for debugging)
gelf git message --dry-run

# Use specific model temporarily
gelf git message --model gemini-2.0-flash-exp
```

## 🔧 Technical Specifications

### Architecture

- **Target**: Staged changes only (`git diff --staged`)
- **AI Provider**: Vertex AI (Gemini models)
- **Default Model**: gemini-2.5-flash-preview-05-20
- **UI Framework**: Bubble Tea (TUI)
- **CLI Framework**: Cobra

### Project Structure

```
cmd/
├── root.go          # Root command definition
└── git/
    ├── git.go       # Git subcommand group
    ├── commit.go    # git commit subcommand implementation
    └── message.go   # git message subcommand implementation
internal/
├── git/
│   └── diff.go      # Git operations (git diff --staged)
├── ai/
│   └── vertex.go    # Vertex AI integration for message generation
├── ui/
│   └── tui.go       # Bubble Tea TUI implementation
└── config/
    └── config.go    # Configuration management (API keys etc)
main.go             # Application entry point
```

## 🎨 TUI Interface

### Loading Screen
```
┌─────────────────────────────────────────┐
│ ⠙ Generating commit message...         │
│                                         │
│ ████████████████░░░░░░░░░░░░░░ 75%      │
└─────────────────────────────────────────┘
```

### Confirmation Screen
```
┌──────────────────────────────────────────────┐
│ 📝 Generated Commit Message:                │
│ ┌──────────────────────────────────────────┐ │
│ │ feat: add user authentication system    │ │
│ │ with JWT support                         │ │
│ └──────────────────────────────────────────┘ │
│ Commit this message? (y)es / (n)o            │
└──────────────────────────────────────────────┘
```

### Committing Screen
```
┌─────────────────────────────────────────┐
│ ⠙ Committing changes...                │
│                                         │
│ ██████████████████████████████ 100%    │
└─────────────────────────────────────────┘
```

### Success Screen
```
┌─────────────────────────────────────────────────┐
│ ✓ Committed: feat: add user authentication     │
│   system with JWT support                       │
└─────────────────────────────────────────────────┘
```

### Error Screen
```
┌─────────────────────────────────────────┐
│ ✗ Error: No staged changes found       │
└─────────────────────────────────────────┘
```

The interface features:
- **Cyan colored** loading messages with animated spinners
- **Blue progress bars** with percentage indicators
- **Rounded border frames** for all states
- **Background colored boxes** for commit messages
- **Color-coded states**: Cyan for loading, Blue for confirmation, Green for success, Red for errors
- **Icon prefixes**: 📝 for messages, ✓ for success, ✗ for errors

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
  default_model: string  # Gemini model to use (default: gemini-2.5-flash-preview-05-20)
```

### Environment Variables

| Variable | Description | Default Value | Required |
|----------|-------------|---------------|----------|
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account key file | - | ✅ |
| `VERTEXAI_PROJECT` | Google Cloud project ID | - | ✅ |
| `VERTEXAI_LOCATION` | Vertex AI location | `us-central1` | ❌ |

**Note**: Model configuration is only available through configuration files.

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
go build                  # Build the project
go test ./...             # Run tests
go mod tidy               # Tidy dependencies
go run main.go git commit    # Run in development
go run main.go git message   # Run message generation only
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