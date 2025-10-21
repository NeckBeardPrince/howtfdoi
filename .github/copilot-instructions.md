# GitHub Copilot Instructions

This file provides guidance to GitHub Copilot when working on the `howtfdoi` repository.

## Project Overview

`howtfdoi` is a command-line tool that provides instant answers to CLI questions, powered by Claude AI from Anthropic. It's designed to be a fast, user-friendly alternative to searching documentation or Stack Overflow.

**Key characteristics:**
- Single-file architecture (`main.go`) for simplicity and portability
- Go-based CLI tool with zero external runtime dependencies
- Uses Claude Haiku model with streaming and prompt caching
- Platform-aware (macOS, Linux, Windows)
- Interactive REPL mode available

## Architecture

### Single-File Design Philosophy

The entire application is intentionally contained in `main.go`. This design choice prioritizes:
- Easy distribution (single binary)
- Simple code navigation
- Minimal dependencies
- Quick compilation

**Important:** Do not split the code into multiple files unless absolutely necessary and discussed with maintainers.

### Core Components

1. **Config Management (`setupConfig`)**: API key loading, history file setup, platform detection
2. **Query Execution (`runQuery`)**: Anthropic API streaming with prompt caching
3. **Response Display (`displayResponse`)**: Color-coded output formatting
4. **Safety Checks (`isDangerous`)**: Pattern matching for risky commands
5. **Interactive Mode (`runInteractiveMode`)**: REPL with readline support

## Building and Testing

### Build Commands

```bash
# Standard build
go build -o howtfdoi

# Build with version info
make build

# Install to GOPATH/bin
go install
# or
make install
```

### Testing

```bash
# Run tests (if present)
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
# or
make test

# Manual testing
./howtfdoi tarball a directory
./howtfdoi -c list files
./howtfdoi -e grep
./howtfdoi -x ls
./howtfdoi  # Interactive mode
```

### Linting and Formatting

```bash
# Run linter
golangci-lint run
# or
make lint

# Format code
go fmt ./...
goimports -w .
# or
make fmt

# Pre-commit hooks
pre-commit run --all-files
```

## Code Style Guidelines

### Go Standards

- Follow standard Go conventions and idioms
- Use `gofmt` and `goimports` for formatting
- Prefer clear, readable code over clever optimizations
- Keep functions focused and concise
- Use meaningful variable names

### Specific to This Project

1. **Error Handling**: Use color-coded error messages with `color.Red().Fprintf()`
2. **Output Formatting**: Commands in bold green, explanations in white, warnings in yellow
3. **Pattern Matching**: Add dangerous command patterns to `dangerousPatterns` slice
4. **Comments**: Add comments for non-obvious logic, especially in parsing and API calls
5. **Constants**: Define at package level for magic numbers and configuration values

### Comments

- Add package-level comments for exported functions (though most are unexported)
- Explain complex regex patterns inline
- Document any platform-specific behavior
- Keep comments concise and up-to-date

## Dependencies

Current dependencies (minimize additions):
- `github.com/anthropics/anthropic-sdk-go` - Claude API client
- `github.com/fatih/color` - Terminal colors
- `github.com/atotto/clipboard` - Cross-platform clipboard
- `github.com/chzyer/readline` - Interactive REPL

**When adding dependencies:**
1. Ensure they're actively maintained
2. Check license compatibility (prefer MIT/BSD/Apache 2.0)
3. Minimize transitive dependencies
4. Update `go.mod` and `go.sum` properly
5. Document why the dependency is needed

## Feature Development Guidelines

### Adding New Flags

1. Define the flag in `main()` using the `flag` package
2. Update the help text and usage examples
3. Add logic in the appropriate function
4. Update README.md with examples
5. Update CHANGELOG.md in the `[Unreleased]` section

### Modifying Prompts

- System prompts are optimized for brevity and include prompt caching
- Changes should maintain the concise output style
- Test with various query types before committing
- Consider both standard and examples mode (`-e` flag)

### Adding Dangerous Patterns

Add to the `dangerousPatterns` slice:
```go
dangerousPatterns := []string{
    `rm\s+-rf\s+/`,      // Dangerous rm command
    `dd\s+.*of=/dev/`,   // DD to device
    // Add new patterns here
}
```

## Safety and Security Considerations

### Input Validation

- All user input goes to Claude API - no direct command execution without confirmation
- `-x` flag always prompts for confirmation before execution
- Dangerous command detection runs on all outputs

### API Key Handling

- API key must come from `ANTHROPIC_API_KEY` environment variable
- Never log or display the API key
- No fallback to hardcoded keys

### Command Execution

- Use `exec.Command` for shell execution
- Always show commands before execution with `-x` flag
- Prompt for user confirmation
- Handle execution errors gracefully

### History File

- History stored at `~/.howtfdoi_history`
- Contains queries and responses (no sensitive data should be logged)
- Use proper file permissions

## Platform Considerations

The tool detects and adapts to:
- **Darwin (macOS)**: Uses `open`, macOS-specific commands
- **Linux**: Uses `xdg-open`, Linux-specific commands  
- **Windows**: Uses Windows-specific commands

When adding platform-specific features:
1. Use `runtime.GOOS` for detection
2. Test on all supported platforms if possible
3. Document platform limitations in README
4. Handle missing platform-specific tools gracefully

## Testing Strategy

### Manual Testing Checklist

Before submitting changes:
- [ ] Build succeeds: `make build`
- [ ] Basic query works: `./howtfdoi list files`
- [ ] Clipboard copy works: `./howtfdoi -c list files`
- [ ] Examples mode works: `./howtfdoi -e tar`
- [ ] Execute mode prompts: `./howtfdoi -x ls`
- [ ] Interactive mode launches: `./howtfdoi`
- [ ] Dangerous commands are detected
- [ ] History is saved properly
- [ ] Linter passes: `make lint`
- [ ] Pre-commit hooks pass: `pre-commit run --all-files`

### Environment Setup for Testing

```bash
# Set API key (use a test/development key)
export ANTHROPIC_API_KEY='your-test-key-here'

# Clean history before testing
rm ~/.howtfdoi_history

# Run tests
make test
```

## Pull Request Guidelines

When submitting PRs:

1. **Update CHANGELOG.md**: Add changes to the `[Unreleased]` section using the appropriate category (Added, Changed, Fixed, etc.)
2. **Update README.md**: If adding features or changing behavior
3. **Follow commit conventions**: Use conventional commit format (`feat:`, `fix:`, `docs:`, etc.)
4. **Keep changes focused**: One logical change per PR
5. **Test thoroughly**: Run the manual testing checklist
6. **Update documentation**: Keep inline comments and docs in sync with code

## Common Patterns

### Adding a New Command-Line Flag

```go
// 1. Define in main()
myFlag := flag.Bool("m", false, "Description of flag")

// 2. Parse flags
flag.Parse()

// 3. Use in logic
if *myFlag {
    // Implementation
}
```

### Adding Color Output

```go
// Use color package
color.Green("Success message")
color.Yellow("Warning message")
color.Red("Error message")

// For formatted output
color.New(color.FgGreen, color.Bold).Fprintf(os.Stdout, "Command: %s\n", cmd)
```

### Streaming API Responses

```go
// Current pattern uses SDK streaming
stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{...})

for stream.Next() {
    event := stream.Current()
    // Handle event
}
```

## Troubleshooting Development Issues

### Build Failures

- Ensure Go 1.21+ is installed
- Run `go mod tidy` to clean dependencies
- Check `go.mod` and `go.sum` are not corrupted

### Import Errors

- Run `goimports -w .` to fix imports
- Check that all dependencies are in `go.mod`

### Linter Errors

- Fix with `golangci-lint run --fix` where possible
- Refer to `.golangci.json` for enabled rules
- Some rules can be ignored with `//nolint:rulename` (use sparingly)

### Pre-commit Hook Failures

- Run `pre-commit run --all-files` to see all issues
- Fix formatting: `make fmt`
- Check commit message format follows conventional commits

## Release Process

Releases are managed by maintainers:

1. Update CHANGELOG.md with version and date
2. Update version in Makefile
3. Create git tag: `git tag v1.0.x`
4. Push tag: `git push origin v1.0.x`
5. GitHub Actions builds and creates release artifacts

## Additional Resources

- **Main documentation**: See `README.md` for user-facing docs
- **Contributing guide**: See `.github/CONTRIBUTING.md` for contributor guidelines
- **Changelog template**: See `.github/CHANGELOG_TEMPLATE.md` for versioning info
- **Claude-specific notes**: See `CLAUDE.md` for Claude Code AI guidance

## Questions?

Open an issue for questions or discussions about development!
