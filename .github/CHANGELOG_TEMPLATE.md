# Changelog Update Template

Use this template when updating CHANGELOG.md for a new release.

## Categories

Use these categories in order (only include categories with changes):

- **Added** - New features
- **Changed** - Changes in existing functionality
- **Deprecated** - Soon-to-be removed features
- **Removed** - Removed features
- **Fixed** - Bug fixes
- **Security** - Security-related changes

## Version Template

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added

- Feature description (#PR-number)

### Changed

- Change description (#PR-number)

### Fixed

- Bug fix description (#PR-number)

### Security

- Security fix description (#PR-number)
```

## Semantic Versioning Guidelines

Given a version number MAJOR.MINOR.PATCH:

- **MAJOR** (X.0.0) - Incompatible API changes
  - Breaking changes to CLI flags
  - Removal of features
  - Major behavioral changes

- **MINOR** (0.X.0) - Backward-compatible functionality additions
  - New flags or commands
  - New features
  - Non-breaking enhancements

- **PATCH** (0.0.X) - Backward-compatible bug fixes
  - Bug fixes
  - Documentation updates
  - Performance improvements without behavior changes

## Examples

### Major Version (2.0.0)

```markdown
## [2.0.0] - 2025-02-15

### Changed

- **BREAKING**: Renamed `-e` flag to `--examples` for clarity (#45)
- **BREAKING**: Changed history file location from `~/.howtfdoi_history` to `~/.config/howtfdoi/history` (#46)

### Added

- Plugin system for custom integrations (#47)
```

### Minor Version (1.1.0)

```markdown
## [1.1.0] - 2025-01-30

### Added

- `-v`/`--verbose` flag for detailed output (#23)
- Support for custom API endpoints via `ANTHROPIC_API_URL` env var (#24)
- Syntax highlighting for code blocks in responses (#25)

### Fixed

- Fixed clipboard copy on WSL2 (#26)
```

### Patch Version (1.0.1)

```markdown
## [1.0.1] - 2025-01-20

### Fixed

- Fixed panic when API key is invalid (#12)
- Improved error messages for network timeouts (#13)
- Fixed color output on Windows terminals (#14)

### Changed

- Updated dependencies to latest versions (#15)
```

## Workflow

1. Add changes to `[Unreleased]` section as you develop
2. When ready to release, move changes to a new version section
3. Update the version links at the bottom
4. Commit the changelog with the version bump
5. Create a git tag matching the version

## Link Format

Update the bottom of CHANGELOG.md:

```markdown
[unreleased]: https://github.com/NeckBeardPrince/howtfdoi/compare/vX.Y.Z...HEAD
[X.Y.Z]: https://github.com/NeckBeardPrince/howtfdoi/compare/vX.Y.Z-1...vX.Y.Z
[X.Y.Z-1]: https://github.com/NeckBeardPrince/howtfdoi/releases/tag/vX.Y.Z-1
```
