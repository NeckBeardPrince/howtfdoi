# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`howtfdoi` is a CLI tool that provides instant command-line answers powered by Claude AI. Users ask natural language questions about CLI commands and get immediate, concise answers with syntax highlighting.

## Building & Testing

```bash
# Build the binary
go build -o howtfdoi

# Install to Go bin directory
go install

# Test basic functionality
./howtfdoi tarball a directory

# Test with flags
./howtfdoi -c list files          # Copy to clipboard
./howtfdoi -e tar                 # Show examples
./howtfdoi -x list files          # Execute with confirmation

# Test interactive mode
./howtfdoi                        # Launches REPL

# Check history
cat ~/.howtfdoi_history
```

## Architecture

### Single-File Design

The entire application is in `main.go` - this is intentional for simplicity and ease of distribution. All features are self-contained.

### Core Flow

1. **Argument Parsing**: Flags (`-c`, `-e`, `-x`) parsed with `flag` package
2. **Query Processing**: Natural language query sent to Claude API (Haiku model)
3. **Response Parsing**: Separates command from explanation for color formatting
4. **Post-Processing**: Safety checks, history logging, clipboard copy, execution, alias suggestions

### Key Components

**Config Management** (`setupConfig`)

- API key from `ANTHROPIC_API_KEY` env var
- History file at `~/.howtfdoi_history`
- Platform detection via `runtime.GOOS`

**Query Execution** (`runQuery`)

- Uses Anthropic SDK streaming for real-time responses
- **Prompt caching enabled** on system prompt for speed
- Platform info passed in query for OS-specific answers

**Response Display** (`displayResponse`)

- Commands: bold green
- Explanations: white
- Color library: `github.com/fatih/color`

**Safety Features**

- `isDangerous()`: Regex patterns for risky commands (rm -rf, dd, etc.)
- `executeCommand()`: Always asks for confirmation before running

**Interactive Mode** (`runInteractiveMode`)

- Uses `github.com/chzyer/readline` for REPL
- Supports inline flags within queries
- Exit with "exit" or "quit"

## System Prompt Strategy

Two prompt modes:

1. **Standard mode**: Concise answers, command + brief explanation
2. **Examples mode** (`-e` flag): 3-5 practical use cases

Both prompts include platform info and are optimized for brevity. The system prompt has prompt caching enabled to reduce latency and cost on repeated queries.

## Dependencies

- `github.com/anthropics/anthropic-sdk-go` - Claude API client
- `github.com/fatih/color` - Terminal colors
- `github.com/atotto/clipboard` - Cross-platform clipboard
- `github.com/chzyer/readline` - Interactive REPL

## Environment Requirements

- `ANTHROPIC_API_KEY` environment variable must be set
- Terminal with ANSI color support for best experience
- Clipboard support requires platform-specific tools (xclip/xsel on Linux)

## Feature Flags

- `-c` Copy command to clipboard
- `-e` Show multiple examples (changes prompt strategy)
- `-x` Execute command with confirmation prompt

## History Format

Stored at `~/.howtfdoi_history`:

```
[YYYY-MM-DD HH:MM:SS] query text
response text
---
```

## Response Parsing Logic

The parser assumes:

- First non-empty line is the command
- Remaining lines are explanation
- Handles cases where Claude returns unstructured text

## Adding New Dangerous Patterns

Update the `dangerousPatterns` slice with regex strings. The checker runs on all responses and displays yellow warnings.

## Platform-Specific Behavior

The tool detects `darwin`, `linux`, `windows` via `runtime.GOOS` and includes platform context in queries. Claude adapts commands accordingly (e.g., `open` vs `xdg-open`).
