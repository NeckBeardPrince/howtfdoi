# howtfdoi

**Instant coding answers via the command line** - powered by Claude

Forget scrolling through Stack Overflow or man pages. Just ask.

## Features

- 🎨 **Color-coded output** - Commands in green, explanations in gray
- 📋 **Copy to clipboard** - One flag to copy commands instantly
- ⚡ **Direct execution** - Run commands with confirmation
- 📚 **Show examples** - Get multiple usage examples
- 🚀 **Interactive mode** - REPL for continuous queries
- 💾 **Query history** - Saves all queries to `~/.howtfdoi_history`
- 🖥️ **Platform-aware** - Detects your OS for tailored answers
- ⚠️ **Danger warnings** - Highlights risky commands in yellow
- 💡 **Smart suggestions** - Offers shell aliases for complex commands
- ⚡ **Blazing fast** - Uses prompt caching for speed

## Installation

```bash
# Build the binary
go build -o howtfdoi

# Optional: Install to your PATH
go install

# Or install directly
go install github.com/neckbeardprince/howtfdoi@latest
```

## Setup

### Option 1: Anthropic Claude (Cloud)

Set your Anthropic API key:

```bash
export ANTHROPIC_API_KEY='your-api-key-here'
```

Get your API key from: <https://console.anthropic.com/>

### Option 2: LM Studio (Local)

Use LM Studio to run models locally with full privacy:

```bash
# Set the provider to LM Studio
export LLM_PROVIDER='lmstudio'

# Optional: Set custom base URL (defaults to http://localhost:1234/v1)
export LLM_BASE_URL='http://localhost:1234/v1'

# Optional: Set the model name (use whatever model you have loaded in LM Studio)
export LLM_MODEL='local-model'
```

1. Download and install [LM Studio](https://lmstudio.ai/)
2. Load any compatible model in LM Studio
3. Start the local server in LM Studio (usually runs on port 1234)
4. Run `howtfdoi` - it will connect to your local LM Studio instance

### Option 3: OpenAI (Cloud)

Use OpenAI's API:

```bash
export LLM_PROVIDER='openai'
export OPENAI_API_KEY='your-openai-api-key'
export LLM_MODEL='gpt-4'  # or gpt-3.5-turbo, etc.
```

## Usage

### Basic Usage

```bash
howtfdoi <your question>
```

### Examples

```bash
# Basic query
howtfdoi tarball a directory
# Output: tar -czf archive.tar.gz directory/
#         (Creates a compressed tarball)

# Copy to clipboard
howtfdoi -c find large files
# Command is copied to clipboard automatically

# Show multiple examples
howtfdoi -e grep
# Shows 5-7 practical grep examples

# Execute directly (with confirmation)
howtfdoi -x list files
# Runs 'ls' after confirmation

# Interactive mode
howtfdoi
# Enters REPL - type questions continuously
```

### Flags

- `-c` - Copy command to clipboard
- `-e` - Show multiple examples
- `-x` - Execute command directly (asks for confirmation)

### Interactive Mode

Run `howtfdoi` without arguments to enter interactive mode:

```bash
$ howtfdoi
🚀 Interactive mode - Type your questions or 'exit' to quit
Tip: Use -c to copy, -x to execute, -e for examples

howtfdoi> find files modified today
find . -mtime -1

howtfdoi> -c search for text recursively
grep -r "text" .
📋 Copied to clipboard!

howtfdoi> exit
Goodbye! 👋
```

## Features in Detail

### 🎨 Color Output

Commands are displayed in **bold green**, explanations in gray. Warnings and dangerous commands appear in yellow/red for visibility.

### ⚠️ Dangerous Command Detection

Automatically warns you about potentially dangerous commands:

- `rm -rf /` or `rm -rf *`
- `dd` operations on devices
- `mkfs` filesystem creation
- Fork bombs and other risky patterns

### 💡 Alias Suggestions

For complex commands (>40 chars or multiple pipes), howtfdoi suggests creating a shell alias:

```bash
$ howtfdoi find all log files and grep for errors
find . -name "*.log" -type f -exec grep -H "ERROR" {} \;

💡 This command is complex. Want to create a shell alias?
Suggested alias:
  alias findalllogfiles='find . -name "*.log" -type f -exec grep -H "ERROR" {} \;'

Add this to your ~/.bashrc or ~/.zshrc
```

### 💾 Query History

All queries are saved to `~/.howtfdoi_history` with timestamps:

```
[2025-01-15 14:30:22] tarball a directory
tar -czf archive.tar.gz directory/
---
```

View your history anytime:

```bash
cat ~/.howtfdoi_history
```

### 🖥️ Platform Detection

Automatically detects your OS (macOS, Linux, Windows) and provides platform-specific commands when relevant.

### 🏠 LM Studio Support

Run completely locally with full privacy:

- **No internet required** - All processing happens on your machine
- **Free** - No API costs, unlimited queries
- **Private** - Your queries never leave your computer
- **Flexible** - Choose any model that runs in LM Studio

Perfect for:
- Sensitive work environments
- Offline development
- Cost-conscious users
- Privacy-focused workflows

## Example Queries

```bash
# File operations
howtfdoi compress a folder
howtfdoi extract tar.gz file
howtfdoi find files by name

# Git operations
howtfdoi undo last commit
howtfdoi show git branch history
howtfdoi cherry pick a commit

# System info
howtfdoi check disk space
howtfdoi show running processes
howtfdoi monitor system resources

# Text processing
howtfdoi replace text in files
howtfdoi count lines in a file
howtfdoi sort and remove duplicates

# Network
howtfdoi check open ports
howtfdoi download file from url
howtfdoi test network connection
```

## Why howtfdoi?

- **Blazing fast**: Uses Claude's Haiku model with streaming + prompt caching
- **Natural language**: Ask questions the way you think
- **CLI focused**: Specialized for command-line tools
- **No browser needed**: Everything in your terminal
- **Smart features**: Copy, execute, examples, history - all built-in
- **Safe**: Warns about dangerous commands before you run them

## Advanced Tips

### Combine Flags

```bash
# Get examples and copy the first one
howtfdoi -e -c tar

# Show examples and execute one
howtfdoi -e -x list processes
```

### Quick Access

Add a shorter alias to your shell config:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias h='howtfdoi'
alias hc='howtfdoi -c'
alias hx='howtfdoi -x'
alias he='howtfdoi -e'
```

Then use:

```bash
h find large files
hc compress directory
he grep
```

## How it Works

1. Takes your natural language question
2. Sends to configured LLM provider (Claude, OpenAI, or LM Studio)
3. Streams response for speed
4. Platform detection ensures OS-specific answers
5. Parses and colorizes the output
6. Checks for dangerous patterns
7. Saves to history automatically

## Environment Variables Reference

Configure `howtfdoi` with these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROVIDER` | Provider to use: `anthropic`, `openai`, or `lmstudio` | Auto-detect based on API keys |
| `ANTHROPIC_API_KEY` | API key for Anthropic Claude | - |
| `OPENAI_API_KEY` | API key for OpenAI | - |
| `LLM_BASE_URL` | Custom API endpoint (for LM Studio or OpenAI-compatible servers) | `http://localhost:1234/v1` for LM Studio |
| `LLM_MODEL` | Model name to use | Provider-specific default |

### Examples

```bash
# Use Anthropic Claude (default if ANTHROPIC_API_KEY is set)
export ANTHROPIC_API_KEY='sk-...'

# Use OpenAI
export LLM_PROVIDER='openai'
export OPENAI_API_KEY='sk-...'
export LLM_MODEL='gpt-4'

# Use LM Studio (local)
export LLM_PROVIDER='lmstudio'
# LLM_BASE_URL and LLM_MODEL will default to sensible values

# Use custom OpenAI-compatible server
export LLM_PROVIDER='openai'
export LLM_BASE_URL='https://my-server.com/v1'
export OPENAI_API_KEY='...'
export LLM_MODEL='my-model'
```

## Troubleshooting

**Colors not showing?**

- Make sure your terminal supports ANSI colors
- Try running `export TERM=xterm-256color`

**Clipboard not working?**

- macOS: Should work out of the box
- Linux: Install `xclip` or `xsel`
- Windows: WSL should work automatically

**API errors?**

- Anthropic: Verify your API key with `echo $ANTHROPIC_API_KEY` and check your account has credits
- OpenAI: Verify your API key with `echo $OPENAI_API_KEY` and check your account has credits
- LM Studio: Ensure LM Studio is running and the server is started (check the server tab in LM Studio)

**LM Studio not connecting?**

- Make sure LM Studio's local server is running (typically on port 1234)
- Check that a model is loaded in LM Studio
- Verify the base URL is correct: `echo $LLM_BASE_URL` (should be `http://localhost:1234/v1`)
- Try accessing the API directly: `curl http://localhost:1234/v1/models`

## Contributing

Issues and PRs welcome! This is a simple tool but there's always room for improvement.

## License

MIT

---

Made with ⚡ and Claude
