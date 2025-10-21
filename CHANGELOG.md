# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.4] - 2025-10-21

### Added

- Verbose logging mode with `-v` flag (shows data directory location and history save confirmations)
- XDG Base Directory specification support for state files
  - History now stored in `$XDG_STATE_HOME/howtfdoi/` (or `~/.local/state/howtfdoi/`)
  - Auto-creates directory structure on first run
- Enhanced help output with `--help` and `-h` flags now showing version information, usage, and examples
- Improved code documentation with detailed function comments

### Changed

- **BREAKING**: `-v` flag now enables verbose mode instead of showing version (use `--version` for version info)
- Refactored response handling into centralized `handleResponse()` function to reduce code duplication
- Pre-compiled dangerous command regex patterns for better performance
- Improved `parseResponse()` logic for cleaner parsing
- Error handling now fails explicitly instead of silently when home directory is unavailable
- History save failures now logged in verbose mode instead of failing silently

### Fixed

- Home directory fallback now provides clear error messages instead of silently falling back to current directory
- Invalid flags now display helpful usage information along with version details

### Performance

- Pre-compiled regex patterns eliminate repeated compilation overhead in `isDangerous()` checks
- Reduced redundant code execution through function consolidation

## [1.0.3] - 2025-10-15

### Added

- Version output with `-v` and `--version` flags
- Repository URL displayed in version output for easy access to downloads and documentation

## [1.0.2] - 2025-10-15

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

[unreleased]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.4...HEAD
[1.0.4]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/NeckBeardPrince/howtfdoi/releases/tag/v1.0.0
