# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **LM Studio Support**: Added support for local AI using LM Studio
  - New `lmstudio` provider option alongside `anthropic` and `openai`
  - `LMStudioProvider` implementation using OpenAI-compatible API
  - Environment variables: `LMSTUDIO_BASE_URL` (default: http://localhost:1234/v1) and `LMSTUDIO_MODEL` (default: local-model)
  - Config file fields: `lmstudio_base_url` and `lmstudio_model`
  - First-run setup wizard now includes LM Studio as option 3
  - Completely local, privacy-focused AI option with no API costs
  - Works offline after model download
  - Verbose mode shows LM Studio base URL and model when provider is lmstudio

### Changed

- Updated help text and documentation to include LM Studio setup instructions
- Provider selection now supports `lmstudio` in addition to `anthropic`, `openai`, and `chatgpt`
- Config struct now includes `LMStudioBaseURL` and `LMStudioModel` fields
- FileConfig struct now includes `LMStudioBaseURL` and `LMStudioModel` YAML fields

## [1.0.8] - 2026-02-11

### Fixed

- **Version display for `go install`**: `howtfdoi --version` now shows the correct version when installed via `go install github.com/neckbeardprince/howtfdoi@latest`. Removed hardcoded "dev" default; uses `runtime/debug.ReadBuildInfo()` as fallback when ldflags aren't set, shows "unknown" only as last resort.

## [1.0.7] - 2026-02-11

### Fixed

- **Version display for `go install`**: `howtfdoi --version` now shows the correct version when installed via `go install github.com/neckbeardprince/howtfdoi@latest`. Uses `runtime/debug.ReadBuildInfo()` as fallback when ldflags aren't set.

## [1.0.6] - 2026-02-11

### Added

- **XDG Config File Support**: API keys and provider preference now persist in a YAML config file
  - Config stored at `$XDG_CONFIG_HOME/howtfdoi/howtfdoi.yaml` (default `~/.config/howtfdoi/howtfdoi.yaml`)
  - Environment variables still take precedence over config file values
  - Config file created with `0600` permissions and config directory with `0700` for security
- **First-Run Setup Wizard**: Interactive prompt on first run when no API key is found
  - Guides users through provider selection (Anthropic or OpenAI)
  - Provides clickable links to developer dashboards for quick API key generation
  - Saves configuration automatically to the config file
- **Automatic `.gitignore`**: A `.gitignore` file is created in the config directory to prevent accidental commits of API keys
- Config and state directories are both created on first run
- Help output now shows config file location and documents `XDG_CONFIG_HOME` / `XDG_STATE_HOME` environment variables
- Verbose mode (`-v`) now logs config file path

### Changed

- **History file renamed**: From `.howtfdoi_history` to `history.log` (now at `$XDG_STATE_HOME/howtfdoi/history.log`)
- Error messages for missing API keys now reference the config file as an alternative to environment variables
- Getting started section in help text updated to mention first-run setup

### Removed

- **Alias suggestions**: Removed the shell alias suggestion feature (`shouldSuggestAlias`, `suggestAlias`, `generateAliasName`)
  - The feature was unreliable when the AI returned non-command responses
  - Removed `aliasLengthThreshold`, `aliasPipeThreshold` constants and `nonAlphanumericRegex`

### Dependencies

- Added `gopkg.in/yaml.v3` for YAML config file parsing

## [1.0.5] - 2025-10-21

### Added

- **Multiple AI Provider Support**: Choose between Claude (Anthropic) or ChatGPT (OpenAI)
  - New `HOWTFDOI_AI_PROVIDER` environment variable to select provider (anthropic, openai, or chatgpt)
  - Auto-detection: Uses Anthropic by default, falls back to OpenAI if only OpenAI key is set
  - Provider abstraction layer with `Provider` interface for extensibility
  - `AnthropicProvider` implementation with streaming and prompt caching
  - `OpenAIProvider` implementation with streaming support (uses GPT-4o-mini model)
- Enhanced help output with comprehensive getting started guide
  - Clear step-by-step setup instructions
  - Better organized sections (Getting Started, Usage, Flags, Environment Variables, Examples)
  - Direct links to API key signup pages (console.anthropic.com, platform.openai.com)
  - More realistic example queries
  - Interactive mode documentation in help text

### Changed

- Help message now includes detailed onboarding for new users
- Error messages now indicate which provider's API key is missing
- Verbose mode now displays which AI provider is being used
- Updated all documentation (README.md, CLAUDE.md) to reflect multi-provider support

### Dependencies

- Added `github.com/sashabaranov/go-openai` v1.41.2 for OpenAI API integration

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

[1.0.8]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.7...v1.0.8
[1.0.7]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/NeckBeardPrince/howtfdoi/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/NeckBeardPrince/howtfdoi/releases/tag/v1.0.0
