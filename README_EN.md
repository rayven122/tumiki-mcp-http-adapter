# Tumiki MCP HTTP Adapter

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/rayven122/tumiki-mcp-http-adapter/workflows/CI/badge.svg)](https://github.com/rayven122/tumiki-mcp-http-adapter/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/rayven122/tumiki-mcp-http-adapter)](https://github.com/rayven122/tumiki-mcp-http-adapter/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/rayven122/tumiki-mcp-http-adapter)](https://goreportcard.com/report/github.com/rayven122/tumiki-mcp-http-adapter)

**Languages**: [🇯🇵 日本語](README.md) | **English**

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Usage Examples](#usage-examples)
- [Command-Line Options](#command-line-options)
- [Development](#development)
- [License](#license)

---

## Overview

A Go-based HTTP proxy server that exposes stdio-based MCP (Model Context Protocol) servers as HTTP endpoints.

### Key Features

- ✅ **Lightweight**: Simple stdio proxy implementation
- ✅ **Quick Start**: Run with just the `--stdio` flag
- ✅ **Dynamic Configuration**: Set environment variables and arguments via HTTP headers (streamable HTTP support)
- ✅ **Custom Header Mapping**: Completely flexible header names for environment variables and arguments

> **📖 Technical Details**: For in-depth information on system architecture, component design, and security design, see [docs/DESIGN_EN.md](docs/DESIGN_EN.md).

---

## Installation

### Pre-built Binaries (Recommended)

Download the latest release from [GitHub Releases](https://github.com/rayven122/tumiki-mcp-http-adapter/releases).

#### macOS / Linux

```bash
# Auto-download and install the latest version
curl -sL https://github.com/rayven122/tumiki-mcp-http-adapter/releases/latest/download/tumiki-mcp-http_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv tumiki-mcp-http /usr/local/bin/

# Verify installation
tumiki-mcp-http --help
```

#### Windows

Download the Windows zip file from the [Releases page](https://github.com/rayven122/tumiki-mcp-http-adapter/releases) and extract it.

### Go Install

```bash
# If Go is installed
go install github.com/rayven122/tumiki-mcp-http-adapter/cmd/tumiki-mcp-http@latest

# Verify installation
tumiki-mcp-http --help
```

**Note**: Ensure `$GOPATH/bin` (typically `~/go/bin`) is in your PATH.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/rayven122/tumiki-mcp-http-adapter.git
cd tumiki-mcp-http-adapter

# Build
task build

# Optionally, add binary to PATH
sudo cp bin/tumiki-mcp-http /usr/local/bin/

# Verify installation
tumiki-mcp-http --help
```

---

## Usage Examples

### Filesystem Server

```bash
# Start server
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-filesystem /Users/yourname/Documents"

# Test
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### GitHub Server (with Environment Variables)

```bash
# Start server
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-github" \
  --env "GITHUB_TOKEN=ghp_xxxxx"

# Test
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "search_repositories",
      "arguments": {"query": "language:go"}
    }
  }'
```

### Header Mapping (Dynamic Configuration)

Dynamically set environment variables and command arguments from HTTP request headers.

**Step 1: Define mapping during CLI startup**

```bash
tumiki-mcp-http --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id" \
  --header-arg "X-Channel=channel"
```

**Step 2: Pass actual values in HTTP requests**

```bash
curl -X POST http://localhost:8080/mcp \
  -H "X-Slack-Token: xoxp-xxxxx" \
  -H "X-Team-Id: T123" \
  -H "X-Channel: general" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

**Result**:

```bash
# Command executed
npx -y server-slack --team-id T123 --channel general

# Environment variable set
SLACK_TOKEN=xoxp-xxxxx
```

---

## Command-Line Options

### Option List

| Option                      | Description                                            | Required | Multiple | Default |
| --------------------------- | ------------------------------------------------------ | -------- | -------- | ------- |
| `--stdio <command>`         | MCP server command to run in stdio mode                | ✅       | ❌       | -       |
| `--port <port>`             | Server port                                            | ❌       | ❌       | `8080`  |
| `--env <KEY=VALUE>`         | Default environment variables                          | ❌       | ✅       | -       |
| `--header-env <HEADER=ENV>` | HTTP header to environment variable mapping            | ❌       | ✅       | -       |
| `--header-arg <HEADER=ARG>` | HTTP header to command argument mapping                | ❌       | ✅       | -       |
| `--log-level <level>`       | Log level (debug/info/warn/error, default: info)       | ❌       | ❌       | `info`  |

### Configuration via Environment Variables

Server startup settings can also be specified via environment variables.

| Environment Variable | Description  | Default   |
| -------------------- | ------------ | --------- |
| `HOST`               | Server host  | `0.0.0.0` |

```bash
# Example: Set host via environment variable, port via command-line option
HOST=127.0.0.1 tumiki-mcp-http --port 3000 --stdio "npx -y server-filesystem /data"
```

---

## Development

For details on development environment setup, testing, and coding conventions, refer to:

**[Development Guide (docs/DEVELOPMENT_EN.md)](docs/DEVELOPMENT_EN.md)**

---

## License

MIT License - See [LICENSE](LICENSE) for details
