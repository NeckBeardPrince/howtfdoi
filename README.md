# howtfdoi

**Instant coding answers via the command line** - powered by AI

Forget scrolling through Stack Overflow or man pages. Just ask.

Ask CLI questions in plain English and get instant answers powered by Claude or ChatGPT.

## Features

- 🤖 **Multiple AI providers** - Choose between Claude (Anthropic) or ChatGPT (OpenAI)
- 🎨 **Color-coded output** - Commands in green, explanations in gray
- 📋 **Copy to clipboard** - One flag to copy commands instantly
- ⚡ **Direct execution** - Run commands with confirmation
- 📚 **Show examples** - Get multiple usage examples
- 🚀 **Interactive mode** - REPL for continuous queries
- 💾 **Query history** - XDG-compliant storage in `~/.local/state/howtfdoi/`
- 🖥️ **Platform-aware** - Detects your OS for tailored answers
- ⚠️ **Danger warnings** - Highlights risky commands in yellow
- 💡 **Smart suggestions** - Offers shell aliases for complex commands
- 🔍 **Verbose mode** - Debug and troubleshoot with detailed logging
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

Choose your AI provider and set the corresponding API key:

### Option 1: Claude (Anthropic) - Default

```bash
export ANTHROPIC_API_KEY='your-api-key-here'
```

Get your API key from: <https://console.anthropic.com/>

### Option 2: ChatGPT (OpenAI)

```bash
export OPENAI_API_KEY='your-api-key-here'
```

Get your API key from: <https://platform.openai.com/>

### Choosing a Provider

By default, `howtfdoi` uses Claude (Anthropic). To use OpenAI/ChatGPT instead:

```bash
# Set the provider explicitly
export HOWTFDOI_AI_PROVIDER=openai

# Or use it inline
HOWTFDOI_AI_PROVIDER=openai howtfdoi list files

# "chatgpt" is an alias for "openai"
HOWTFDOI_AI_PROVIDER=chatgpt howtfdoi list files
```

If you don't set `HOWTFDOI_AI_PROVIDER`, the tool will:
1. Use Anthropic if `ANTHROPIC_API_KEY` is set
2. Fall back to OpenAI if only `OPENAI_API_KEY` is set
3. Use OpenAI if both keys are set but you specify the provider

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

# Verbose mode (shows data directory, history saves)
howtfdoi -v compress files
# Displays: Using data directory: ~/.local/state/howtfdoi

# Show version
howtfdoi --version
# Output: howtfdoi version 1.0.4

# Show help
howtfdoi --help
# Shows usage, flags, and examples

# Interactive mode
howtfdoi
# Enters REPL - type questions continuously
```

### Flags

- `-c` - Copy command to clipboard
- `-e` - Show multiple examples
- `-v` - Enable verbose logging (shows data directory, history saves)
- `-x` - Execute command directly (asks for confirmation)
- `--version` - Show version information
- `--help` / `-h` - Show usage help and examples

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

All queries are saved with timestamps following the XDG Base Directory specification:

**Default location:** `~/.local/state/howtfdoi/.howtfdoi_history`

```
[2025-01-15 14:30:22] tarball a directory
tar -czf archive.tar.gz directory/
---
```

View your history anytime:

```bash
cat ~/.local/state/howtfdoi/.howtfdoi_history
```

**Custom location:** Set `XDG_STATE_HOME` to change the base directory:

```bash
export XDG_STATE_HOME=/custom/path
howtfdoi find files
# History saved to: /custom/path/howtfdoi/.howtfdoi_history
```

### 🔍 Verbose Mode

Use the `-v` flag to enable detailed logging for debugging and troubleshooting:

```bash
$ howtfdoi -v find large files
Using data directory: /Users/you/.local/state/howtfdoi
find / -type f -size +100M -exec ls -lh {} \;
(Finds files larger than 100MB and lists them with sizes)
Saved to history: /Users/you/.local/state/howtfdoi/.howtfdoi_history
```

Verbose mode shows:
- Data directory location on startup
- History file save confirmations
- Warnings if history cannot be saved

### 🖥️ Platform Detection

Automatically detects your OS (macOS, Linux, Windows) and provides platform-specific commands when relevant.

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

- **Blazing fast**: Uses fast models (Claude Haiku or GPT-4o-mini) with streaming + prompt caching
- **Flexible**: Choose your preferred AI provider (Claude or ChatGPT)
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
alias hv='howtfdoi -v'
alias hx='howtfdoi -x'
alias he='howtfdoi -e'
```

Then use:

```bash
h find large files
hc compress directory
hv debug issue          # verbose mode
he grep
```

## How it Works

1. Takes your natural language question
2. Determines which AI provider to use (Claude or ChatGPT)
3. Streams to the selected API using fast models (Haiku or GPT-4o-mini)
4. Uses prompt caching for repeated queries (even faster with Claude!)
5. Platform detection ensures OS-specific answers
6. Parses and colorizes the output
7. Checks for dangerous patterns
8. Saves to history automatically

## Troubleshooting

**Colors not showing?**

- Make sure your terminal supports ANSI colors
- Try running `export TERM=xterm-256color`

**Clipboard not working?**

- macOS: Should work out of the box
- Linux: Install `xclip` or `xsel`
- Windows: WSL should work automatically

**API errors?**

- Verify your API key is set:
  - For Claude: `echo $ANTHROPIC_API_KEY`
  - For ChatGPT: `echo $OPENAI_API_KEY`
- Check which provider is being used: `howtfdoi -v list files`
- Verify your account has credits (Anthropic Console or OpenAI Dashboard)

## Contributing

Issues and PRs welcome! This is a simple tool but there's always room for improvement.

## License

MIT

---

Made with ⚡ and Claude
