# Tumiki MCP HTTP Adapter - Development Guide

**Languages**: [🇯🇵 日本語](DEVELOPMENT.md) | **English**

## Development Environment Setup

### Required Tools

- **Go 1.25+** - Programming language
- **[Task](https://taskfile.dev/)** - Task runner
- **[golangci-lint](https://golangci-lint.run/)** - Go linter
- **[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)** - Import statement organizer

### Installing Development Tools

```bash
# Auto-install development tools
task install-tools
```

---

## Development Commands

Check available tasks:

```bash
task --list
```

### Main Commands

| Command          | Description                                    |
|-----------------|------------------------------------------------|
| `task build`    | Build binary                                   |
| `task test`     | Run tests                                      |
| `task coverage` | Run tests with coverage                        |
| `task fmt`      | Format code                                    |
| `task lint`     | Run linter                                     |
| `task check`    | Run all checks (format, vet, lint, test)       |
| `task clean`    | Remove build artifacts                         |

---

## Running Without Building

During development, you can run directly without building:

```bash
# Run directly without building
go run ./cmd/tumiki-mcp-http --stdio "npx -y @modelcontextprotocol/server-filesystem /data"

# Run with environment variables
go run ./cmd/tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-github" \
  --env "GITHUB_TOKEN=ghp_xxxxx"

# Run with header mapping
go run ./cmd/tumiki-mcp-http \
  --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id"
```

---

## Testing

### Running Unit Tests

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# With race detector
go test -race ./...

# Specific package only
go test ./internal/proxy
go test ./internal/process
```

### Coverage Reports

```bash
# Measure coverage
task coverage

# Generate HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Testing Policy

For detailed testing policy, see [CLAUDE.md](../CLAUDE.md).

- ✅ Target 100% coverage for testable functions
- ✅ Test all normal cases, error cases, and edge cases
- ✅ Test case names in "input_condition_expected_result" format
- ✅ Always test error handling

---

## Coding Conventions

This project follows these conventions:

### Formatting

- **gofmt**: Go's standard formatter
- **goimports**: Auto-organize import statements

```bash
task fmt
```

### Linting

- **golangci-lint**: Comprehensive static analysis tool (configuration: `.golangci.yml`)

```bash
task lint
```

### Testing

- **Coverage target**: 100% for testable functions
- **Table-driven tests**: Structured multiple test cases
- **Details**: See [CLAUDE.md](../CLAUDE.md)

### Error Handling

- Explicitly return errors
- Use `log.Fatal` minimally (main function only)
- Error messages should be specific and clear

---

## Pre-Commit Checks

Always run the following before committing:

```bash
task check
```

This command executes:

1. Code formatting (`gofmt`, `goimports`)
2. Static analysis (`go vet`)
3. Linter (`golangci-lint`)
4. All tests

---

## Project Structure

```text
tumiki-mcp-http-adapter/
├── cmd/tumiki-mcp-http/     # Main application
│   ├── main.go              # Entry point
│   └── main_test.go         # CLI tests
├── internal/                 # Private packages
│   ├── proxy/               # HTTP server, header parsing
│   │   ├── server.go
│   │   └── server_test.go
│   └── process/             # Process execution
│       ├── executor.go
│       └── executor_test.go
├── docs/                     # Documentation
│   ├── DESIGN.md            # Design document (Japanese)
│   ├── DESIGN_EN.md         # Design document (English)
│   ├── DEVELOPMENT.md       # Development guide (Japanese)
│   └── DEVELOPMENT_EN.md    # Development guide (English)
├── .golangci.yml            # Linter configuration
├── Taskfile.yml             # Task definitions
├── go.mod                    # Go module definition
├── CLAUDE.md                # Development guidelines (test policy)
├── README.md                # Project overview (Japanese)
└── README_EN.md             # Project overview (English)
```

---

## Troubleshooting

### Test Failures

```bash
# Show detailed error information
go test -v ./...

# Detect concurrency issues with race detector
go test -race ./...
```

### Linter Errors

```bash
# Fix auto-fixable issues
task fmt

# Show detailed linter information
golangci-lint run --verbose
```

### Build Failures

```bash
# Clean dependencies
go mod tidy

# Clear build cache
go clean -cache

# Rebuild
task build
```

---

## Release

This project uses [GoReleaser](https://goreleaser.com/) and GitHub Actions for automated releases.

### Release Process

#### 1. Release Preparation

```bash
# Fetch latest main branch
git checkout main
git pull origin main

# Ensure all tests pass
task check
```

#### 2. Create Version Tag

```bash
# Create tag following semantic versioning
# MAJOR.MINOR.PATCH (e.g., v1.0.0, v1.2.3)

# Create tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag to GitHub
git push origin v1.0.0
```

#### 3. Automated Build & Release

Pushing a tag triggers GitHub Actions to automatically:

1. **Cross-platform build**: Generate binaries for macOS/Linux/Windows
2. **Create archives**: Compress in tar.gz (Unix) and zip (Windows) formats
3. **Generate checksums**: Create sha256 checksum files
4. **Create GitHub Release**: Publish binaries with release notes

#### 4. Verify Release

Check that the release is published on the [Releases page](https://github.com/rayven122/tumiki-mcp-http-adapter/releases).

### Versioning

Follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible functionality additions
- **PATCH**: Backward-compatible bug fixes

**Examples**:
- `v1.0.0`: First stable release
- `v1.1.0`: New features added
- `v1.1.1`: Bug fixes
- `v2.0.0`: Major update with breaking changes

### Pre-releases

Create tags in the following format for beta or release candidate versions:

```bash
# Beta version
git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"

# Release candidate
git tag -a v1.0.0-rc.1 -m "Release candidate v1.0.0-rc.1"

git push origin <tag>
```

GoReleaser automatically treats these as pre-releases.

### Release Configuration

Release settings are managed in these files:

- **[.goreleaser.yaml](../.goreleaser.yaml)**: GoReleaser configuration
- **[.github/workflows/release.yml](../.github/workflows/release.yml)**: GitHub Actions workflow

---

## References

- **[README.md](../README.md)** - Project overview and usage (Japanese)
- **[README_EN.md](../README_EN.md)** - Project overview and usage (English)
- **[DESIGN.md](DESIGN.md)** - System architecture and design (Japanese)
- **[DESIGN_EN.md](DESIGN_EN.md)** - System architecture and design (English)
- **[CLAUDE.md](../CLAUDE.md)** - Test policy and coding conventions
- **[Taskfile.yml](../Taskfile.yml)** - Detailed task definitions
- **[.golangci.yml](../.golangci.yml)** - Detailed linter configuration
- **[GoReleaser Documentation](https://goreleaser.com/)** - Official GoReleaser documentation
