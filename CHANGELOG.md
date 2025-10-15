# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## 1.0.2 - 2025-10-15

### Added

- Support for Haiku 4.5

## [1.0.1] - 2025-10-08

### Fixed

- Fixed module path in go.mod

## [1.0.0] - 2025-10-08

### Added

- Support for Claude AI (Haiku model) via Anthropic API
- Initial release of howtfdoi CLI tool
- Natural language command-line queries powered by Claude AI (Haiku model)
- Real-time streaming responses for instant answers
- Color-coded output (bold green for commands, white for explanations)
- `-c` flag to copy commands to clipboard automatically
- `-x` flag to execute commands directly with safety confirmation
- `-e` flag to show multiple practical examples (3-5 use cases)
- Interactive REPL mode - run without arguments for continuous queries
- Query history tracking saved to `~/.howtfdoi_history` with timestamps
- Platform detection (macOS, Linux, Windows) for OS-specific answers
- Dangerous command detection with warnings (rm -rf, dd, mkfs, etc.)
- Smart alias suggestions for complex commands (>40 chars or multiple pipes)
- Prompt caching for faster repeated queries and reduced API costs
- Cross-platform clipboard support (macOS, Linux with xclip/xsel, Windows WSL)
- Inline flag support in interactive mode
- GitHub Actions CI workflow with linting, testing, and cross-compilation
- GitHub Actions release workflow with automatic binary publishing
- Pre-commit hooks configuration with Go formatting and linting
- Comprehensive README with usage examples and troubleshooting
- CLAUDE.md for AI-assisted development guidance
- Contributing guidelines and development documentation
- Makefile with common development tasks
- golangci-lint configuration
- Dependabot configuration for automatic dependency updates

### Security

- Private key detection in pre-commit hooks
- Dangerous command pattern detection with user warnings
- Confirmation prompts before command execution
- API key validation on startup

[unreleased]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/NeckBeardPrince/howtfdoi/releases/tag/v1.0.0
