# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Klama is an AI-powered CLI tool for troubleshooting DevOps issues, specifically focusing on Kubernetes debugging. It uses an interactive TUI (Terminal User Interface) built with Bubble Tea to provide a conversational interface where users can ask questions and receive AI-powered guidance with optional command execution.

## Development Guidelines

- **Never push directly to main branch**
- If changes are made to the main branch, always:
  - Create a new branch for changes
  - Open a Pull Request (PR)
  - Ask for review before merging
  - Wait for explicit permission to merge

## Architecture

The project follows a clean architecture pattern with these key components:

- **cmd/**: CLI command structure using Cobra framework
  - `root.go`: Main CLI setup with global flags (`--config`, `--debug`)
  - `k8s.go`: Kubernetes debugging command that initializes the full application stack
- **config/**: Configuration management using Viper
  - Supports multiple config file locations (XDG, legacy home directory)
  - Handles OpenAI-compatible API configuration with pricing
- **internal/agent/**: AI agent abstraction layer
  - Defines system prompts for different agent types (currently only Kubernetes)
  - Handles structured JSON responses with retry logic for malformed responses
- **internal/llm/**: Language model client implementation
  - OpenAI-compatible API client with conversation history management
  - Token usage tracking and cost calculation
- **internal/executer/**: Command execution interface
  - Terminal command execution with validation
  - Currently focused on safe kubectl commands only
- **internal/ui/**: Interactive TUI using Bubble Tea
  - Conversational interface with viewport for chat history
  - Command confirmation workflow for safety
- **internal/logger/**: Debug logging system

## Common Commands

### Development
```bash
# Run the application
go run main.go k8s

# Run with debug logging
go run main.go k8s --debug

# Run tests
go test ./...

# Run tests for specific package
go test ./internal/agent

# Build binary
go build -o klama main.go
```

### Configuration
The application requires a config file with AI model settings:
```yaml
agent:
  name: "gpt-4o-mini"
  base_url: "https://api.openai.com/v1"
  auth_token: ""  # Set via KLAMA_AGENT_TOKEN env var
```

## Key Implementation Details

### Agent System
- Agents are typed (currently only `AgentTypeKubernetes`)
- All responses must be valid JSON with specific structure: `{"answer": string, "run_command": string, "reason_for_command": string}`
- Built-in retry mechanism for malformed JSON responses (3 attempts)
- Conversation history maintained throughout session

### Command Execution Safety
- Only read-only kubectl commands are allowed (get, describe, logs)
- All commands validated before execution
- User confirmation required for every command
- Commands limited to 4-hour log lookups by default

### UI State Management
- Four main states: `StateTyping`, `StateAsking`, `StateExecuting`, `StateWaitingForConfirmation`
- Keyboard shortcuts: Ctrl+C (exit), Ctrl+R (restart), Ctrl+S (toggle command response visibility)
- Mouse scrolling and keyboard navigation supported

### Configuration Management
- Supports XDG Base Directory specification
- Environment variable override for auth token (`KLAMA_AGENT_TOKEN`)
- Automatic default config generation if none exists

## Testing

Tests are located alongside source files with `_test.go` suffix. Key test areas:
- Agent JSON parsing and validation
- Configuration loading and validation
- LLM client functionality
- Terminal command execution
- UI state transitions

## Build and Release

### Release Workflow Overview

The project uses an automated release pipeline with GoReleaser and GitHub Actions:

#### CI/CD Pipeline

**Continuous Integration** (`.github/workflows/build-test.yaml`):
- **Triggers**: Pull requests to `main` branch
- **Jobs**:
  - `fmt_and_vet`: Code formatting and vetting checks (Go 1.22, Ubuntu)
  - `unit_tests`: Test matrix across Ubuntu and Windows with Go 1.22
- **Environment**: `CGO_ENABLED=0` for static builds

**Release Pipeline** (`.github/workflows/release.yaml`):
- **Trigger**: Git tag push (`tags: ['*']`)
- **Process**: Uses GoReleaser Action v6 with `goreleaser release --clean`
- **Requirements**: `PERSONAL_ACCESS_TOKEN` secret for GitHub operations

#### Version Management

- Version stored in `cmd/root.go` with defaults: `version = "dev"`, `arch = "dev"`
- Build-time injection via ldflags: `-X github.com/eliran89c/klama/cmd.version={{.Version}} -X github.com/eliran89c/klama/cmd.arch={{.Arch}}`

#### Supported Platforms

- **Linux**: amd64, arm64
- **macOS (Darwin)**: amd64, arm64
- **Windows**: amd64 (arm64 excluded)
- **Archive formats**: tar.gz (Unix-like), zip (Windows)

#### Release Artifacts

Each release includes:
- Source code archives
- Cross-platform binaries
- Checksums file
- Automated Homebrew formula publication

#### Distribution Channels

1. **GitHub Releases**: Direct binary downloads
2. **Homebrew**: `brew install eliran89c/tap/klama`
3. **Go Install**: `go install github.com/eliran89c/klama@latest`

#### Commit Message Guidelines

**IMPORTANT**: Always use conventional commit prefixes for proper changelog generation.

**Commit Message Format**: `<type>: <description>`

**Required Prefixes** (for changelog inclusion):
- `feat:` - New features
- `fix:` - Bug fixes  
- `chore:` - Maintenance tasks, dependency updates, refactoring
- `BREAKING CHANGE:` - Breaking changes

**Other Prefixes** (not included in changelog but good practice):
- `docs:` - Documentation changes
- `test:` - Test additions or modifications
- `ci:` - CI/CD pipeline changes
- `refactor:` - Code refactoring without feature changes

#### Dependabot PR Commit Mapping

**When merging Dependabot PRs, ALWAYS create a proper commit message:**

```bash
# Instead of using the default merge message, use:
gh pr merge <PR_NUMBER> --squash --body "chore: bump <package> from <old_version> to <new_version>"
```

**Examples**:
```
chore: bump github.com/charmbracelet/bubbletea from 1.3.4 to 1.3.5
chore: bump github.com/spf13/cobra from 1.8.1 to 1.9.1
chore: bump github.com/spf13/viper from 1.20.0 to 1.20.1
```

#### Changelog Configuration

GoReleaser automatically generates changelogs from commits with these prefixes:
- `feat:` - New features
- `fix:` - Bug fixes
- `chore:` - Maintenance tasks (including dependency updates)
- `BREAKING CHANGE:` - Breaking changes

#### Creating a Release

```bash
# Create and push a new tag
git tag v0.x.x
git push origin v0.x.x

# GitHub Actions will automatically:
# 1. Run GoReleaser
# 2. Build cross-platform binaries
# 3. Create GitHub release
# 4. Update Homebrew tap
# 5. Generate changelog
```

#### Homebrew Integration

- **Repository**: `eliran89c/homebrew-tap`
- **Formula location**: `Formula/klama.rb`
- **Test command**: `system "#{bin}/klama version"`
- **Automatic updates**: On each release

## Dependabot PR Merge Workflow

When handling Dependabot PRs, follow this systematic approach for safe dependency updates:

### 1. Discovery and Assessment
```bash
# List all open Dependabot PRs
gh pr list --author=app/dependabot

# Review each PR for necessity and safety
gh pr view <PR_NUMBER>
```

### 2. Pre-merge Checks
Before starting work on any Dependabot PR:
- Check for merge conflicts with the base branch
- Ensure all CI checks are passing
- Verify the PR branch is up-to-date with main

### 3. Changelog Review and Impact Analysis
For each dependency update, review the changelog to understand potential impacts:

```bash
# First, check the Dependabot PR body for changelog information
gh pr view <PR_NUMBER>
# Dependabot usually includes changelog/release notes in the PR description

# If changelog not in PR body, check the dependency's repository:
# For GitHub-hosted dependencies:
# Visit: https://github.com/owner/repo/releases
# Or: https://github.com/owner/repo/blob/main/CHANGELOG.md

# Look for:
# - Breaking changes that might affect our code
# - New features we could utilize
# - Bug fixes that might impact our functionality
# - Security patches
# - Deprecation warnings
```

**If breaking changes or significant updates are found:**
- Document the required code changes
- Assess the impact on our codebase
- Present findings to the user with:
  - Summary of changes needed
  - Justification for the changes
  - Potential impact on functionality
- **Wait for user approval before proceeding with merge**

### 4. Sequential Merge Process
Merge PRs one at a time to avoid conflicts and enable safe rollback:

```bash
# For each PR in sequence:
# 1. Check for conflicts before starting
gh pr view <PR_NUMBER>

# 2. If conflicts exist, resolve them:
gh pr checkout <PR_NUMBER>
git fetch origin main
git merge origin/main
# Resolve conflicts in go.mod and go.sum manually
go mod tidy
git add .
git commit -m "Resolve merge conflicts for <dependency> upgrade"
git push origin HEAD:<branch_name>

# 3. Wait for CI to complete
gh pr checks <PR_NUMBER> --watch

# 4. Merge only after all checks pass with proper commit message
gh pr merge <PR_NUMBER> --squash --body "chore: bump <package> from <old_version> to <new_version>"

# 5. Test the merged changes
go test ./...
go build -o klama-test .
./klama-test version
```

### 5. Post-merge Validation
After each merge:
- Run full test suite: `go test ./...`
- Build test binary: `go build -o klama-test .`
- Test basic functionality: `./klama-test version`
- Verify no regressions in core functionality

### 6. Important Notes
- Always use `klama-test` as the test binary name (already in .gitignore)
- Some PRs may be automatically merged by GitHub if they're simple updates
- Skip PRs that are already closed or auto-merged
- If multiple PRs update the same dependency, only process the latest one
- Commit changes between each PR merge for safe rollback capability

### 7. Common Merge Conflict Resolution
When resolving conflicts in go.mod:
- Keep the higher version number for the target dependency
- Use the latest toolchain version from main
- Use the latest versions of other dependencies from main
- Run `go mod tidy` after manual resolution
- Ensure all tests pass before pushing the resolution

### 8. CI Requirements
- Never merge until all CI checks pass
- Wait for all status checks (fmt_and_vet, unit tests on all platforms)
- Use `gh pr checks --watch` to monitor CI progress
- If CI fails, investigate and fix issues before merging