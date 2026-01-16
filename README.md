<div align="center">
  <img src="logo.svg" alt="Kairos logo" width="200" height="200">
  <h1>Kairos</h1>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
    <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
    <img src="https://img.shields.io/badge/Status-Alpha-yellow?style=for-the-badge" alt="Status">
  </p>

  > *Where time tracking meets artificial intelligence. A CLI companion that remembers, analyzes, and evolves with your work patterns.*

</div>

---

## Table of Contents

- [What is Kairos?](#what-is-kairos)
- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Commands](#commands)
- [AI Integration](#ai-integration)
- [MCP Server](#mcp-server)
- [Configuration](#configuration)
- [Data Storage](#data-storage)
- [Visualization](#visualization)
- [Building from Source](#building-from-source)
- [Performance](#performance)
- [Tech Stack](#tech-stack)
- [Contributing](#contributing)
- [License](#license)
- [Name Origin](#name-origin)

---

## What is Kairos?

Kairos is a sophisticated command-line tool for tracking work hours, designed with a unique blend of simplicity and intelligence. Named after the Greek concept of the "right moment" or "opportune time," Kairos goes far beyond basic time tracking.

At its core, Kairos understands that **time is not just a metric—it's a narrative**. Every clock-in, every break, every session tells a story about how you work. Kairos listens to these stories, learns from them, and provides insights that help you work smarter, not just harder.

### The Philosophy

Unlike traditional time trackers that merely record numbers, Kairos embraces four core principles:

1. **Think** - Analyze patterns, reason about decisions, and understand the "why" behind your work habits
2. **Evolve** - Continuously improve based on data-driven insights and self-reflection
3. **Consciousness** - Maintain self-awareness about your current state, goals, and progress
4. **Persist** - Remember important insights, deadlines, and context across sessions and time

---

## Features

### Core Time Tracking

- **Clock In/Out** - Seamless start and end of work sessions with optional notes
- **Break Management** - Automatic break deduction from work hours
- **Session Notes** - Attach context to your work sessions
- **Session Editing** - Modify past entries when needed

### Progress Tracking

- **Daily View** - See today's work at a glance
- **Weekly Summary** - Automatic consolidation of weekly hours with goal tracking
- **Monthly Statistics** - Long-term trends and averages
- **Goal Progress** - Track progress toward configurable weekly goals (default: 38.5 hours)

### AI-Powered Insights

- **Natural Language Questions** - Ask questions like "Can I leave now?" or "How many hours do I need today?"
- **Predictive Analysis** - AI forecasts when you'll reach your weekly goal
- **Pattern Recognition** - Identifies trends in your work habits
- **Smart Recommendations** - Personalized suggestions based on your history

### Visualization

- **SVG Charts** - Beautiful vector graphics for weekly and monthly overviews
- **HTML Reports** - Shareable, interactive reports
- **Progress Bars** - Visual goal tracking
- **Daily Breakdowns** - Detailed session analysis

### MCP Server

A powerful Model Context Protocol server that enables AI assistants to interact with your work data:

| Tool | Description |
|------|-------------|
| `think` | Deep reasoning about work patterns and scheduling |
| `evolve` | Self-improvement suggestions based on your data |
| `consciousness` | Self-awareness about your current work state |
| `persist` | Long-term memory storage and retrieval |

### Privacy First

- **100% Local** - All data stored locally, no cloud sync
- **SQLite Database** - Efficient, reliable, and portable
- **Your Data, Your Control** - Export, backup, and restore anytime

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/sriinnu/kairos.git
cd kairos
CGO_ENABLED=1 go build -o kairos ./cmd/kairos

# Start your first session
./kairos clockin "Working on feature X"

# Clock in with a specific time (forgot to clock in earlier)
./kairos clockin -t "08:45" "Morning work"

# Check your progress
./kairos status

# End your session (with 30 min break)
./kairos clockout 30

# List all sessions with UUIDs
./kairos sessions

# Edit the current session's note
./kairos edit -n "Updated note"

# Edit a specific session (use partial UUID from sessions list)
./kairos edit a052c6e0 -t "09:00" -n "Corrected start time"

# See weekly summary
./kairos week

# Ask AI a question
./kairos ask "Can I leave early today?"

# Start MCP server for AI assistants
./kairos mcp start
```

### Session Identification

Sessions use UUIDs for identification. The `sessions` command shows the first 8 characters:

```
a052c6e0: Jan 15 09:00 (active) - Working on project [ACTIVE]
```

Use partial or full UUID with `edit` and `delete`:

```bash
./kairos edit a052c6e0 -t "08:30"          # Edit by partial UUID
./kairos edit a052c6e0-1984-47b1-...       # Edit by full UUID
./kairos delete a052c6e0                   # Delete session
```

---

## Installation

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/dl/)
- **SQLite3** - Usually comes with Go's mattn driver
- **Ollama** (optional) - For AI features [Install Ollama](https://ollama.com)

### From Source

```bash
git clone https://github.com/sriinnu/kairos.git
cd kairos

# Build with SQLite support (requires gcc)
CGO_ENABLED=1 go build -o kairos ./cmd/kairos

# Optional: Install to PATH
# Linux/macOS:
sudo cp kairos /usr/local/bin/

# Windows:
copy kairos.exe C:\Windows\System32\
```

### Verify Installation

```bash
kairos --help
kairos version
```

---

## Commands

### Time Tracking

| Command | Aliases | Flags | Description |
|---------|---------|-------|-------------|
| `clockin [note]` | `in`, `ci` | `-t HH:MM` | Start a work session |
| `clockout [minutes]` | `out`, `co` | `-t HH:MM, -b minutes` | End current session |
| `status` | `st`, `today` | | Show today's progress |
| `week [date]` | `w` | | Weekly summary |
| `month` | `m` | | Monthly statistics |

### Session Management

| Command | Aliases | Flags | Description |
|---------|---------|-------|-------------|
| `sessions` | `ls`, `list` | | List recent sessions with UUIDs |
| `edit [uuid]` | `e`, `update` | `-t HH:MM, -n note, -b minutes` | Edit session |
| `delete <uuid>` | `del`, `rm`, `remove` | `-f` | Delete a session |
| `batch <cmd>` | `bulk` | `--ids, --date, --dry-run` | Batch operations |

### AI & Analysis

| Command | Aliases | Description |
|---------|---------|-------------|
| `ask "question"` | `a`, `ai` | Ask AI about your hours |
| `predict` | | AI goal completion prediction |
| `analyze` | | AI work pattern analysis |

### Configuration & Utilities

| Command | Description |
|---------|-------------|
| `config` | Show current configuration |
| `completion [shell]` | Generate shell completion (bash/zsh/fish/powershell) |
| `archive` | Archive old months to markdown |
| `history` | Show historical summary |

### Visualization

| Command | Description |
|---------|-------------|
| `visualize week` | Generate weekly SVG |
| `visualize month` | Generate monthly SVG |
| `visualize html` | Generate HTML report |

### MCP Server

| Command | Description |
|---------|-------------|
| `mcp start` | Start MCP server |
| `mcp tools` | List MCP tools |
| `mcp query <tool>` | Query tool directly |
| `mcp register` | Print client config |

### Configuration

| Command | Description |
|---------|-------------|
| `config` | Show current settings |

### Shell Completion

Enable tab completion for your shell:

```bash
# Bash (Linux)
kairos completion bash > /etc/bash_completion.d/kairos

# Zsh
kairos completion zsh > "${fpath[1]}/_kairos"

# Fish
kairos completion fish > ~/.config/fish/completions/kairos.fish

# PowerShell
kairos completion powershell > kairos.ps1
. kairos.ps1
```

---

## AI Integration

Kairos supports multiple AI providers for intelligent, context-aware responses.

### Supported Providers

| Provider | Type | Setup |
|----------|------|-------|
| **Ollama** | Local | Install [Ollama](https://ollama.com), run `ollama serve` |
| **OpenAI** | Cloud | Set `OPENAI_API_KEY` environment variable |
| **Claude** | Cloud | Set `ANTHROPIC_API_KEY` environment variable |
| **Gemini** | Cloud | Set `GEMINI_API_KEY` environment variable |

### Setup Ollama (Recommended for Privacy)

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama server
ollama serve

# Pull a model (recommended: llama3.2)
ollama pull llama3.2
```

### Configure AI Provider

```bash
# Set provider in config (edit ~/.kairos.yaml)
ai_provider: ollama  # or openai, claude, gemini

# For cloud providers, set environment variables
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"
export GEMINI_API_KEY="your-key-here"
```

### AI Commands

```bash
# Ask contextual questions
kairos ask "Can I leave now?"
kairos ask "How many hours should I work tomorrow?"
kairos ask "Am I on track for my weekly goal?"

# Get predictions
kairos predict

# Analyze work patterns
kairos analyze
```

### AI-Powered MCP Tools

When connected to AI assistants via MCP:

```json
{
  "tool": "think",
  "arguments": {
    "question": "Should I take a break?",
    "analysis_type": "productivity"
  }
}
```

```json
{
  "tool": "evolve",
  "arguments": {
    "timeframe": "week",
    "focus_area": "consistency"
  }
}
```

```json
{
  "tool": "consciousness",
  "arguments": {
    "aspect": "all"
  }
}
```

---

## MCP Server

Kairos includes a powerful **Model Context Protocol (MCP)** server that enables AI assistants like Claude, Cursor, and others to interact with your work data.

### Starting the Server

```bash
# Start on default port (8765)
kairos mcp start

# Start on custom port
kairos mcp start -p 9000
```

### Connecting AI Assistants

```bash
# Get connection configuration
kairos mcp register

# Output:
# {
#   "mcpServers": {
#     "kairos": {
#       "url": "http://localhost:8765/mcp",
#       "transport": "http"
#     }
#   }
# }
```

### Available MCP Tools

#### Think
Deep reasoning and analysis about work patterns.

```json
{
  "tool": "think",
  "arguments": {
    "question": "Should I take a break?",
    "analysis_type": "productivity",
    "include_history": true
  }
}
```

#### Evolve
Self-improvement suggestions based on your data.

```json
{
  "tool": "evolve",
  "arguments": {
    "timeframe": "week",
    "focus_area": "productivity"
  }
}
```

Returns: evolution score, suggestions, daily targets.

#### Consciousness
Self-awareness about your current work state.

```json
{
  "tool": "consciousness",
  "arguments": {
    "aspect": "all"
  }
}
```

Returns: is_working, hours_today, hours_week, goal_progress, recommendations.

#### Persist
Long-term memory storage and retrieval.

```json
{
  "tool": "persist",
  "arguments": {
    "action": "store",
    "key": "project-alpha",
    "value": "Client deadline: March 15",
    "category": "projects",
    "tags": ["client", "deadline"]
  }
}
```

**Persist Actions:**
- `store` - Store a new memory
- `retrieve` - Get a memory by key
- `search` - Search memories
- `list` - List all memories
- `update` - Update existing memory
- `delete` - Delete a memory
- `cleanup` - Remove old memories

### Direct CLI Queries

```bash
# Query tools directly from CLI
kairos mcp query consciousness aspect=current
kairos mcp query think question="Should I take a break?" analysis_type=productivity
kairos mcp query persist action=list
kairos mcp query persist action=store key="reminder" value="Team meeting at 3pm" category="meetings"
```

---

## Configuration

Kairos uses `~/.kairos.yaml` for configuration (created automatically on first run).

### Default Configuration

```yaml
# Database location
database_path: ~/.kairos/data.db

# Weekly goal in hours (38.5 is standard in Austria)
weekly_goal: 38.5

# Ollama settings
ollama_url: http://localhost:11434
ollama_model: llama3.2

# MCP server port
mcp_port: 8765
```

### Configuration Commands

```bash
# View current configuration
kairos config

# Update a setting (if implemented)
kairos config set weekly_goal 40.0
kairos config set ollama_model llama3.3
```

---

## Data Storage

All data is stored locally in SQLite. Your privacy is protected—no cloud sync, no external servers.

### Database Location

- **Linux/macOS:** `~/.kairos/data.db`
- **Windows:** `%USERPROFILE%\.kairos\data.db`

### Database Schema

```
work_sessions    - Individual work sessions
daily_summary    - Daily aggregations
weekly_summary   - Weekly aggregations
monthly_summary  - Monthly aggregations
memories         - Long-term memories (MCP persist)
```

### Backup and Export

```bash
# Copy the database file
cp ~/.kairos/data.db backup.db

# Export to CSV (future feature)
kairos export csv -o work-hours.csv
```

---

## Visualization

Generate visual reports of your work patterns.

### SVG Output

```bash
# Weekly overview chart
kairos visualize week

# Monthly overview chart
kairos visualize month
```

Outputs scalable vector graphics suitable for embedding or viewing in browsers.

### HTML Reports

```bash
# Generate interactive HTML report
kairos visualize html -o report.html

# Open in browser (macOS)
open report.html

# Linux
xdg-open report.html

# Windows
start report.html
```

HTML reports include:
- Progress bars
- Daily breakdowns
- Statistics
- Responsive design

---

## Building from Source

### Prerequisites

- Go 1.21 or higher
- Git

### Build Steps

```bash
# Clone the repository
git clone https://github.com/sriinnu/kairos.git
cd kairos

# Download dependencies
go mod download

# Build
go build -o kairos ./cmd/kairos

# Run tests
go test ./...

# Build with race detection
go build -race -o kairos-race ./cmd/kairos
```

### Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o kairos-linux-amd64 ./cmd/kairos

# macOS
GOOS=darwin GOARCH=amd64 go build -o kairos-darwin-amd64 ./cmd/kairos

# Windows
GOOS=windows GOARCH=amd64 go build -o kairos-windows-amd64.exe ./cmd/kairos
```

---

## Performance

Kairos is engineered for performance and reliability.

### Design Principles

- **Zero Memory Leaks** - Rigorous context handling and resource cleanup
- **Efficient Queries** - Indexed SQLite tables for fast lookups
- **Concurrent Safety** - Mutex-protected database access
- **Minimal Footprint** - Lightweight dependencies, fast startup
- **Resource Conscious** - Background operations don't block CLI

### Benchmarks

Typical operations complete in milliseconds:

| Operation | Expected Time |
|-----------|---------------|
| Clock In/Out | < 10ms |
| Status Query | < 20ms |
| Weekly Summary | < 50ms |
| Month Statistics | < 100ms |

### Memory Usage

- Idle: ~5-10 MB
- During operations: ~10-20 MB
- MCP server: ~15-25 MB

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| CLI Framework | Cobra |
| Database | SQLite3 |
| AI Integration | Ollama (HTTP) |
| Protocol | MCP (Model Context Protocol) |
| Configuration | YAML |
| Visualization | SVG, HTML |

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Ways to Contribute

- Report bugs
- Suggest features
- Improve documentation
- Add tests (see `internal/*/*_test.go`)
- Submit pull requests

### Development Setup

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Verify build: `go build -o kairos ./cmd/kairos`
6. Submit a Pull Request

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

Sriinnu - [@sriinnu](https://github.com/sriinnu)

## Name Origin

**Kairos** (καιρός) is an ancient Greek word with a rich history.

While **Chronos** refers to chronological, sequential time (the kind we measure with clocks), **Kairos** represents the "right moment," the "opportune time," or a "season." It speaks to the qualitative, meaningful moments rather than mere quantities.

In Greek mythology, Kairos was depicted as a winged youth with a razor or scales—representing the fleeting, critical nature of the "right moment" that must be seized when it appears.

A fitting name for a tool that helps you make the most of your working hours, recognizing that time is not just about counting hours, but about finding the right moments to work, rest, and evolve.

---

## Acknowledgments

- [Ollama](https://ollama.com) - For local AI inference
- [Cobra](https://github.com/spf13/cobra) - For the excellent CLI framework
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver
- The open source community

---

<div align="center">

**Built with intention by Sriinnu**

*May you find your Kairos.*

</div>
