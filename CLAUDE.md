# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`howtfdoi` is a CLI tool that provides instant command-line answers powered by AI (Claude, ChatGPT, or LM Studio). Users ask natural language questions about CLI commands and get immediate, concise answers with syntax highlighting.

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
./howtfdoi -v find large files    # Verbose mode (shows data dir, config file, history saves)

# Test version and help
./howtfdoi --version              # Show version info
./howtfdoi --help                 # Show usage, flags, and examples

# Test interactive mode
./howtfdoi                        # Launches REPL (or first-run setup if no API key)

# Test with OpenAI/ChatGPT
export OPENAI_API_KEY="your-key"
HOWTFDOI_AI_PROVIDER=openai ./howtfdoi list files
HOWTFDOI_AI_PROVIDER=chatgpt ./howtfdoi list files  # chatgpt is an alias for openai

# Test with LM Studio (local)
# First, start LM Studio and load a model
HOWTFDOI_AI_PROVIDER=lmstudio ./howtfdoi list files
LMSTUDIO_BASE_URL=http://localhost:1234/v1 LMSTUDIO_MODEL=local-model HOWTFDOI_AI_PROVIDER=lmstudio ./howtfdoi list files

# Check history (XDG Base Directory compliant)
cat ~/.local/state/howtfdoi/history.log

# Test with custom XDG directories
XDG_STATE_HOME=/tmp/test ./howtfdoi list files
XDG_CONFIG_HOME=/tmp/testconfig ./howtfdoi list files

# Test first-run setup (delete config file, unset env vars)
rm ~/.config/howtfdoi/howtfdoi.yaml
ANTHROPIC_API_KEY="" OPENAI_API_KEY="" ./howtfdoi list files
```

## Architecture

### Single-File Design

The entire application is in `main.go` - this is intentional for simplicity and ease of distribution. All features are self-contained.

### Core Flow

1. **Argument Parsing**: Flags (`-c`, `-e`, `-x`, `-v`) parsed with `flag` package
2. **Config Loading**: Loads config file, resolves provider/API key (env vars > config file > first-run setup)
3. **Provider Selection**: Determines AI provider (Anthropic/Claude, OpenAI/ChatGPT, or LM Studio)
4. **Query Processing**: Natural language query sent to selected AI provider via provider abstraction
5. **Response Parsing**: Separates command from explanation for color formatting
6. **Post-Processing**: Consolidated in `handleResponse()` - safety checks, history logging, clipboard copy, execution

### Key Components

**Config Management** (`setupConfig`, `getDataDirectory`, `getConfigDirectory`)

- **Config file**: YAML at `$XDG_CONFIG_HOME/howtfdoi/howtfdoi.yaml` (default `~/.config/howtfdoi/howtfdoi.yaml`)
- **Priority order**: Environment variables > config file values > defaults
- Provider selection via `HOWTFDOI_AI_PROVIDER` env var or `provider` in config file
- API key from env var (`ANTHROPIC_API_KEY` / `OPENAI_API_KEY`) or config file
- LM Studio configuration via `LMSTUDIO_BASE_URL` and `LMSTUDIO_MODEL` env vars or config file
- Auto-detects provider if not explicitly set (checks which API key is available)
- **XDG Base Directory support**: History at `$XDG_STATE_HOME/howtfdoi/history.log` (default `~/.local/state/howtfdoi/history.log`)
- Platform detection via `runtime.GOOS`
- Verbose mode flag controls logging verbosity
- Both config and state directories auto-created on first run

**Config File** (`FileConfig`, `loadConfigFile`, `saveConfigFile`)

- `FileConfig` struct with YAML tags for serialization
- `loadConfigFile()`: Reads and parses YAML config, returns zero-value if missing
- `saveConfigFile()`: Writes config with warning header, creates `.gitignore` alongside to prevent accidental commits
- Config file written with `0600` permissions, config directory with `0700`

**First-Run Setup** (`runFirstTimeSetup`)

- Triggered when no API key is found and stdin is a terminal (`isatty` check)
- Prompts for provider selection (Anthropic, OpenAI, or LM Studio)
- Provides clickable links to developer dashboards for API key generation
- For LM Studio, prompts for base URL and model name with sensible defaults
- Saves configuration via `saveConfigFile()`

**Provider Abstraction** (`Provider`, `AnthropicProvider`, `OpenAIProvider`, `LMStudioProvider`)

- Interface-based design allows switching between AI providers
- `AnthropicProvider`: Uses Anthropic SDK streaming with prompt caching enabled
- `OpenAIProvider`: Uses OpenAI SDK streaming for real-time responses
- `LMStudioProvider`: Embeds `OpenAIProvider` with custom base URL for local LM Studio server
- All providers implement the same `Query` interface for consistency

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
- Handles: display, dangerous command checks, history logging, clipboard copy, execution
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
- `github.com/sashabaranov/go-openai` - OpenAI/ChatGPT API client (also used for LM Studio)
- `github.com/fatih/color` - Terminal colors
- `github.com/atotto/clipboard` - Cross-platform clipboard
- `github.com/chzyer/readline` - Interactive REPL
- `github.com/mattn/go-isatty` - Terminal detection for first-run setup
- `gopkg.in/yaml.v3` - YAML config file parsing

## Environment Requirements

- `ANTHROPIC_API_KEY` environment variable or config file entry (required for Claude/Anthropic)
- `OPENAI_API_KEY` environment variable or config file entry (required for ChatGPT/OpenAI)
- `HOWTFDOI_AI_PROVIDER` environment variable (optional, defaults to anthropic)
- `LMSTUDIO_BASE_URL` environment variable (optional for LM Studio, defaults to http://localhost:1234/v1)
- `LMSTUDIO_MODEL` environment variable (optional for LM Studio, defaults to local-model)
- `XDG_CONFIG_HOME` to override config directory (default: `~/.config`)
- `XDG_STATE_HOME` to override state directory (default: `~/.local/state`)
- Terminal with ANSI color support for best experience
- Clipboard support requires platform-specific tools (xclip/xsel on Linux)
- For LM Studio: LM Studio application running locally with a model loaded

## Config File

Located at `$XDG_CONFIG_HOME/howtfdoi/howtfdoi.yaml` (default `~/.config/howtfdoi/howtfdoi.yaml`):

```yaml
# WARNING: This file contains API keys. Do NOT commit this file to git.
# Add this file to your .gitignore if it is inside a repository.
provider: anthropic
anthropic_api_key: sk-ant-...
openai_api_key: sk-...
# For LM Studio (local):
# provider: lmstudio
# lmstudio_base_url: http://localhost:1234/v1
# lmstudio_model: local-model
```

A `.gitignore` is automatically created in the config directory to ignore `howtfdoi.yaml`.

## Feature Flags

- `-c` Copy command to clipboard
- `-e` Show multiple examples (changes prompt strategy)
- `-x` Execute command with confirmation prompt
- `-v` Enable verbose mode (shows data directory, config file path, AI provider, history save confirmations)
- `--version` Show version information and repository URL
- `--help` / `-h` Show usage, available flags, and examples

## History Format

Stored at `$XDG_STATE_HOME/howtfdoi/history.log` (or `~/.local/state/howtfdoi/history.log` by default):

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
- Constants extracted for magic numbers (`maxTokens`, etc.)
- `handleResponse()` consolidates post-processing logic
- `parseInteractiveLine()` separates parsing from interactive loop
- `getDataDirectory()` encapsulates XDG state directory logic
- `getConfigDirectory()` encapsulates XDG config directory logic

**Error Handling:**
- Explicit error handling with clear user messages
- Verbose mode for debugging and troubleshooting
- No silent failures - all errors logged or reported

**Documentation:**
- Comprehensive function comments explaining purpose and behavior
- Code structure documented for maintainability
