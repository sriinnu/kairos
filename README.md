<div align="center">
  <img src="logo.svg" alt="Kairos logo" width="200" height="200">
  <h1>Kairos</h1>
</div>

A CLI tool for tracking work hours with AI-powered insights. Built in Go with SQLite storage and Ollama integration.

## What is Kairos?

Kairos helps you track and understand your working hours. Named after the Greek god of the "right moment" or "opportune time," it goes beyond simple time tracking by providing:

- **Daily/Weekly/Monthly tracking** with automatic consolidation
- **Progress toward goals** (default: 38.5 hours/week for Austrian work culture)
- **AI-powered answers** to questions like "Can I leave now?" or "How many hours do I need today?"
- **Beautiful visualizations** in SVG and HTML formats

## Installation

```bash
git clone https://github.com/yourusername/kairos.git
cd kairos
go build -o kairos ./cmd/samaya
./kairos install  # Optional: install to PATH
```

## Requirements

- Go 1.21+
- SQLite3
- [Ollama](https://ollama.com) (optional, for AI features)

## Quick Start

```bash
# Clock in to start your workday
kairos clockin

# Clock out when done (optionally with break time)
kairos clockout 30  # 30 minute break

# Check your progress
kairos status
kairos week
kairos month
```

## Commands

| Command | Description |
|---------|-------------|
| `clockin [note]` | Start a work session |
| `clockout [minutes]` | End current session, optional break |
| `status` | Show today's progress |
| `week` | Weekly summary with breakdown |
| `month` | Monthly statistics |
| `ask "question"` | Ask AI about your hours |
| `predict` | AI predicts goal completion |
| `visualize week\|month\|html` | Generate SVG/HTML reports |
| `config` | Show current settings |
| `mcp start` | Start MCP server for AI assistants |
| `mcp tools` | List available MCP tools |
| `mcp query <tool>` | Query MCP tools directly |

## AI Features

Kairos uses local LLMs via Ollama to provide intelligent answers:

```bash
# Ask contextual questions
kairos ask "Can I leave early today?"
kairos ask "How many hours should I work tomorrow?"

# Get predictions
kairos predict
```

Make sure Ollama is running: `ollama serve`

## MCP Server - Versatile AI Integration

Kairos includes a powerful **Model Context Protocol (MCP)** server that enables AI assistants to interact with your work data. The MCP server provides four core capabilities:

### MCP Tools

| Tool | Description |
|------|-------------|
| `think` | Reasoning and analysis about work patterns |
| `evolve` | Self-improvement suggestions based on your data |
| `consciousness` | Self-awareness about your current work state |
| `persist` | Long-term memory storage and retrieval |

### Starting the MCP Server

```bash
# Start on default port 8765
kairos mcp start

# Start on custom port
kairos mcp start -p 9000
```

### Using with AI Assistants

```bash
# Get MCP config for your AI client
kairos mcp register

# Query tools directly
kairos mcp query consciousness aspect=current
kairos mcp query think question="Should I take a break?" analysis_type=productivity
kairos mcp query persist action=list
```

Connect AI assistants (Claude, Cursor, etc.) to: `http://localhost:8765/mcp`

### Persist Tool Operations

```bash
# Store a memory
kairos mcp query persist action=store key="project_alpha" value="Client deadline: March 15" category="projects"

# Retrieve
kairos mcp query persist action=retrieve key="project_alpha"

# Search
kairos mcp query persist action=search query="deadline"

# List all
kairos mcp query persist action=list
```

## Configuration

Kairos uses `~/.kairos.yaml` for configuration:

```yaml
database_path: ~/.kairos/data.db
weekly_goal: 38.5
ollama_url: http://localhost:11434
ollama_model: llama3.2
```

## Visualization

Generate visual reports:

```bash
# SVG output (copy to browser)
kairos visualize week

# Save HTML report
kairos visualize html -o report.html
```

HTML reports include progress bars, daily breakdowns, and statistics.

## Data Storage

All data is stored locally in SQLite (`~/.kairos/data.db`). Your privacy is protected—no cloud sync, no external servers.

## Tech Stack

- **Go** - Core language
- **SQLite** - Local data persistence
- **Cobra** - CLI framework
- **Ollama** - Local AI inference
- **MCP** - Model Context Protocol for AI integration

## Performance

Kairos is built with performance and memory safety in mind:

- **Zero memory leaks** - Proper context handling and resource cleanup
- **Efficient SQLite queries** - Indexed tables and optimized lookups
- **Concurrent-safe operations** - Mutex-protected database access
- **Minimal dependencies** - Lightweight and fast startup

## License

MIT License - see LICENSE file for details.

## Author

Sriinnu - [@yourusername](https://github.com/sriinnu)

## Name Origin

"Kairos" (καιρός) is an ancient Greek word meaning the "right moment," "opportune time," or "season." Unlike Chronos (sequential, quantified time), Kairos represents the qualitative, meaningful moments. A fitting name for a tool that helps you make the most of your working hours.
