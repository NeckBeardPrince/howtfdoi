# Contributing to howtfdoi

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- (Optional) [pre-commit](https://pre-commit.com/) for automated code checks

### Installing Pre-commit Hooks

We use pre-commit hooks to ensure code quality. To set them up:

```bash
# Install pre-commit (if not already installed)
# macOS
brew install pre-commit

# Linux
pip install pre-commit

# Windows
pip install pre-commit

# Install the git hooks
pre-commit install
pre-commit install --hook-type commit-msg
```

Now pre-commit will run automatically on `git commit`. To run manually:

```bash
# Run on all files
pre-commit run --all-files

# Run on staged files only
pre-commit run
```

### Building and Testing

```bash
# Build
go build -o howtfdoi

# Run tests (when added)
go test -v ./...

# Run linter
golangci-lint run

# Format code
go fmt ./...
goimports -w .
```

## Making Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Update CHANGELOG.md (see section below)
5. Run pre-commit checks: `pre-commit run --all-files`
6. Commit with a conventional commit message (e.g., `feat: add new feature`)
7. Push and create a pull request

## Updating the Changelog

For all user-facing changes, update `CHANGELOG.md`:

1. Add your changes to the `[Unreleased]` section
2. Use the appropriate category:
   - **Added** - New features
   - **Changed** - Changes in existing functionality
   - **Deprecated** - Soon-to-be removed features
   - **Removed** - Removed features
   - **Fixed** - Bug fixes
   - **Security** - Security-related changes

3. Include your PR number: `- Feature description (#PR-number)`

Example:
```markdown
## [Unreleased]

### Added

- Support for custom prompts via config file (#42)

### Fixed

- Fixed clipboard paste on Linux (#43)
```

See `.github/CHANGELOG_TEMPLATE.md` for detailed examples and versioning guidelines.

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `refactor:` Code refactoring
- `test:` Adding tests
- `chore:` Maintenance tasks

Example: `feat: add history search command`

## Code Style

- Follow standard Go conventions
- Run `gofmt` and `goimports` before committing (pre-commit does this automatically)
- Keep functions focused and concise
- Add comments for non-obvious logic

## Pull Request Process

1. Update README.md if adding new features
2. Update CHANGELOG.md with your changes in the `[Unreleased]` section
3. Ensure all CI checks pass
4. Request review from maintainers
5. Address any feedback
6. Maintainers will merge once approved

## Questions?

Open an issue for questions or discussions!
