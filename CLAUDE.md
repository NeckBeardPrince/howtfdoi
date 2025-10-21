# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`howtfdoi` is a CLI tool that provides instant command-line answers powered by AI (Claude or ChatGPT). Users ask natural language questions about CLI commands and get immediate, concise answers with syntax highlighting.

## Building & Testing

```bash
# Build the binary
go build -o howtfdoi

# Build with version info (uses Makefile)
make build

# Install to Go bin directory
go install

# Test basic functionality
./howtfdoi tarball a directory

# Test with flags
./howtfdoi -c list files          # Copy to clipboard
./howtfdoi -e tar                 # Show examples
./howtfdoi -x list files          # Execute with confirmation
./howtfdoi -v find large files    # Verbose mode (shows data dir, history saves)

# Test version and help
./howtfdoi --version              # Show version info
./howtfdoi --help                 # Show usage, flags, and examples

# Test interactive mode
./howtfdoi                        # Launches REPL

# Test with OpenAI/ChatGPT
export OPENAI_API_KEY="your-key"
HOWTFDOI_AI_PROVIDER=openai ./howtfdoi list files
HOWTFDOI_AI_PROVIDER=chatgpt ./howtfdoi list files  # chatgpt is an alias for openai

# Check history (XDG Base Directory compliant)
cat ~/.local/state/howtfdoi/.howtfdoi_history

# Test with custom XDG_STATE_HOME
XDG_STATE_HOME=/tmp/test ./howtfdoi list files
```

## Architecture

### Single-File Design

The entire application is in `main.go` - this is intentional for simplicity and ease of distribution. All features are self-contained.

### Core Flow

1. **Argument Parsing**: Flags (`-c`, `-e`, `-x`, `-v`) parsed with `flag` package
2. **Provider Selection**: Determines AI provider (Anthropic/Claude or OpenAI/ChatGPT) based on `HOWTFDOI_AI_PROVIDER` env var and available API keys
3. **Query Processing**: Natural language query sent to selected AI provider via provider abstraction
4. **Response Parsing**: Separates command from explanation for color formatting
5. **Post-Processing**: Consolidated in `handleResponse()` - safety checks, history logging, clipboard copy, execution, alias suggestions

### Key Components

**Config Management** (`setupConfig`, `getDataDirectory`)

- Provider selection via `HOWTFDOI_AI_PROVIDER` env var (anthropic, openai, chatgpt)
- API key from `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` env var depending on provider
- Auto-detects provider if `HOWTFDOI_AI_PROVIDER` not set (checks which API key is available)
- **XDG Base Directory support**: History file at `$XDG_STATE_HOME/howtfdoi/.howtfdoi_history` (or `~/.local/state/howtfdoi/.howtfdoi_history`)
- Platform detection via `runtime.GOOS`
- Verbose mode flag controls logging verbosity
- Directory auto-creation on first run

**Provider Abstraction** (`Provider`, `AnthropicProvider`, `OpenAIProvider`)

- Interface-based design allows switching between AI providers
- `AnthropicProvider`: Uses Anthropic SDK streaming with prompt caching enabled
- `OpenAIProvider`: Uses OpenAI SDK streaming for real-time responses
- Both providers implement the same `Query` interface for consistency

**Query Execution** (`runQuery`)

- Creates appropriate provider based on config
- Platform info passed in query for OS-specific answers
- Streaming responses for speed

**Response Display** (`displayResponse`)

- Commands: bold green
- Explanations: white
- Color library: `github.com/fatih/color`

**Response Handling** (`handleResponse`)

- Centralized post-processing function (reduces code duplication)
- Handles: display, dangerous command checks, history logging, clipboard copy, execution, alias suggestions
- Uses `ResponseOptions` struct for clean flag passing

**Response Parsing** (`parseResponse`, `parseInteractiveLine`)

- `parseResponse()`: Extracts command and explanation from Claude's response
- `parseInteractiveLine()`: Parses interactive mode input with inline flags

**Safety Features**

- `isDangerous()`: **Pre-compiled** regex patterns for risky commands (rm -rf, dd, etc.) - eliminates repeated compilation overhead
- `executeCommand()`: Always asks for confirmation before running
- Dangerous patterns defined at startup for performance

**Interactive Mode** (`runInteractiveMode`)

- Uses `github.com/chzyer/readline` for REPL
- Supports inline flags within queries (`-c`, `-x`, `-e`)
- Exit with "exit" or "quit"

## System Prompt Strategy

Two prompt modes:

1. **Standard mode**: Concise answers, command + brief explanation
2. **Examples mode** (`-e` flag): 3-5 practical use cases

Both prompts include platform info and are optimized for brevity. When using Anthropic, the system prompt has prompt caching enabled to reduce latency and cost on repeated queries.

## Dependencies

- `github.com/anthropics/anthropic-sdk-go` - Claude API client
- `github.com/sashabaranov/go-openai` - OpenAI/ChatGPT API client
- `github.com/fatih/color` - Terminal colors
- `github.com/atotto/clipboard` - Cross-platform clipboard
- `github.com/chzyer/readline` - Interactive REPL

## Environment Requirements

- `ANTHROPIC_API_KEY` environment variable (required for Claude/Anthropic)
- `OPENAI_API_KEY` environment variable (required for ChatGPT/OpenAI)
- `HOWTFDOI_AI_PROVIDER` environment variable (optional, defaults to anthropic)
- Terminal with ANSI color support for best experience
- Clipboard support requires platform-specific tools (xclip/xsel on Linux)

## Feature Flags

- `-c` Copy command to clipboard
- `-e` Show multiple examples (changes prompt strategy)
- `-x` Execute command with confirmation prompt
- `-v` Enable verbose mode (shows data directory location, history save confirmations)
- `--version` Show version information and repository URL
- `--help` / `-h` Show usage, available flags, and examples

## History Format

Stored at `$XDG_STATE_HOME/howtfdoi/.howtfdoi_history` (or `~/.local/state/howtfdoi/.howtfdoi_history` by default):

```
[YYYY-MM-DD HH:MM:SS] query text
response text
---
```

The tool follows the XDG Base Directory specification for state files. Set `XDG_STATE_HOME` to customize the location.

## Response Parsing Logic

The parser assumes:

- First non-empty line is the command
- Remaining lines are explanation
- Handles cases where Claude returns unstructured text

## Adding New Dangerous Patterns

Update the `dangerousPatterns` slice with pre-compiled `*regexp.Regexp` objects. The patterns are compiled once at startup for performance. The checker runs on all responses and displays yellow warnings.

Example:
```go
dangerousPatterns = []*regexp.Regexp{
    regexp.MustCompile(`rm\s+-rf\s+/`),
    regexp.MustCompile(`your-new-pattern-here`),
}
```

## Platform-Specific Behavior

The tool detects `darwin`, `linux`, `windows` via `runtime.GOOS` and includes platform context in queries. Claude adapts commands accordingly (e.g., `open` vs `xdg-open`).

## Code Quality & Performance (v1.0.4+)

Recent refactoring improvements include:

**Performance Optimizations:**
- Pre-compiled regex patterns (eliminates runtime compilation overhead)
- Centralized response handling reduces code duplication

**Code Organization:**
- Constants extracted for magic numbers (`maxTokens`, `aliasLengthThreshold`, etc.)
- `handleResponse()` consolidates post-processing logic
- `parseInteractiveLine()` separates parsing from interactive loop
- `getDataDirectory()` encapsulates XDG directory logic

**Error Handling:**
- Explicit error handling with clear user messages
- Verbose mode for debugging and troubleshooting
- No silent failures - all errors logged or reported

**Documentation:**
- Comprehensive function comments explaining purpose and behavior
- Code structure documented for maintainability
