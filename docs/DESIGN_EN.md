# Tumiki MCP HTTP Adapter - System Architecture Design Document

**Languages**: [🇯🇵 日本語](DESIGN.md) | **English**

## System Architecture

### System Architecture Diagram

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP Request
       │ (Custom Headers: X-Slack-Token, X-Team-Id, etc.)
       ▼
┌─────────────────────────────────────┐
│   Tumiki MCP HTTP Adapter           │
│                                     │
│  ┌──────────────────────────────┐  │
│  │  Proxy Handler               │  │
│  │  - Header Mapping            │  │
│  │  - Env/Args Building         │  │
│  └───────────┬──────────────────┘  │
│              ▼                      │
│  ┌──────────────────────────────┐  │
│  │  Process Executor            │  │
│  │  - Stdio Process Launch      │  │
│  │  - Input/Output Handling     │  │
│  └──────────────────────────────┘  │
└─────────────┬───────────────────────┘
              │
              ▼
      ┌──────────────┐
      │ MCP Server   │
      │ (stdio mode) │
      └──────────────┘
```

### Data Flow

1. **Request Reception**: HTTP POST /mcp
2. **Header Parsing**: Extract environment variables and arguments based on custom mappings
3. **Configuration Merging**: Default environment variables + header-derived values
4. **Process Launch**: Execute MCP server in stdio mode
5. **Response Return**: Return MCP server output as HTTP response

---

## Component Design

### 1. cmd/tumiki-mcp-http

**Responsibilities**: Application entry point, CLI parsing

**Main Features**:

- Define and parse command-line flags
- Build configuration
- Server startup and shutdown handling
- Signal handling (Graceful Shutdown)

**Key Design Points**:

- `ArrayFlags` type for flags that can be specified multiple times
- `parseStdioCommand()` parses shell-style command strings (with quote support)
- `buildConfigFromFlags()` constructs configuration from CLI flags
- `startServer()` implements Graceful Shutdown using defer + exitCode pattern

### 2. internal/proxy

**Responsibilities**: HTTP server, MCP endpoint handler, header parsing

**Main Data Structures**:

- **Config**: Server configuration
  - Port number
  - stdio command and arguments
  - Default environment variables
  - Header → environment variable mappings
  - Header → argument mappings

- **Timeout Constants**:
  - ReadTimeout: 30 seconds (HTTP request reading)
  - WriteTimeout: 30 seconds (HTTP response writing)
  - ShutdownTimeout: 5 seconds (Graceful Shutdown)
  - ProcessTimeout: 30 seconds (stdio process execution)

- **Server**: HTTP server instance
  - Configuration (Config)
  - Structured logger
  - HTTP server core

**Public API**:

- `NewServer`: Create server instance
- `Start`: Start server (Context-aware)
- `Handler`: Get HTTP handler (for testing)

**Internal Functions**:

- `handleMCP`: MCP HTTP endpoint handler
- `parseHeaders`: Extract environment variables and arguments from HTTP headers

**Processing Flow (handleMCP)**:

1. Parse headers with `parseHeaders()`
2. Merge with default environment variables
3. Merge arguments (without modifying original slice - appendAssign mitigation)
4. Read request body
5. Execute process (with timeout)
6. Return response (with error handling)

**Key Design Points**:

- `parseHeaders()` implemented as pure function (testability)
- Argument merging without modifying original slice (concurrency safety)
- Error responses returned with appropriate HTTP status codes
- Context propagation for proper cleanup on client disconnection

### 3. internal/process

**Responsibilities**: Launch stdio processes and handle input/output

**Main Data Structures**:

- **Executor**: Process execution management
  - Command name
  - Command arguments
  - Environment variable map
  - Structured logger

**Public API**:

- `NewExecutor`: Create Executor instance
- `Execute`: Execute process and handle input/output (Context-aware)

**Processing Flow (Execute)**:

1. Create process with `exec.CommandContext`
2. Set environment variables
3. Connect stdin/stdout/stderr pipes
4. Start process
5. Asynchronously read stderr (prevent data race with sync.WaitGroup)
6. Write input data to stdin
7. Close stdin
8. Read JSON-RPC response from stdout
9. Wait for process completion
10. Wait for stderr reading completion (WaitGroup.Wait)
11. Error handling (log stderr contents)
12. Return output data

**Key Design Points**:

- **Data Race Prevention**: Synchronize asynchronous stderr reading with `sync.WaitGroup`
- **Resource Management**: Prevent resource leaks with defer or explicit Close()
- **Context Propagation**: Support timeout/cancellation with `exec.CommandContext`
- **Error Logging**: Output stderr contents in structured logs on process failure

---

## Dynamic Header Mapping Design

### Streamable HTTP Pattern

Unlike traditional fixed configuration, this design allows dynamic setting of different environment variables and arguments for each HTTP request.

### Design Principles

**Mapping Definition (CLI Startup)**:

```
X-Slack-Token → SLACK_TOKEN (environment variable)
X-Team-Id     → team-id (argument)
X-Channel     → channel (argument)
```

**Runtime Conversion (HTTP Request)**:

When receiving the following headers in an HTTP request:
```
X-Slack-Token: xoxp-12345
X-Team-Id: T123
X-Channel: general
```

Converted according to mapping definition:
```
Environment variable: SLACK_TOKEN=xoxp-12345
Arguments: --team-id T123 --channel general

Executed command:
npx -y server-slack --team-id T123 --channel general
```

### Design Benefits

1. **Dynamic Configuration**: Use different tokens and arguments for each request
2. **Multi-tenant Support**: Different authentication credentials per team or user
3. **Security**: Pass sensitive information via environment variables rather than command-line arguments
4. **Flexibility**: Completely free header name definition

---

## Error Handling Design

### HTTP Error Responses

| Status Code               | Purpose        | Occurrence Condition            |
| ------------------------- | -------------- | ------------------------------- |
| 200 OK                    | Normal         | Process execution success       |
| 400 Bad Request           | Invalid request| Body reading failure            |
| 500 Internal Server Error | Server error   | Process execution failure/timeout|

### Logging Design

**Structured Logging (slog)**:

Uses the standard library `slog` package for structured log output.

Log output example:
```
INFO: Server starting addr=0.0.0.0:8080
ERROR: Process execution failed error="..." stderr="..."
DEBUG: Failed to copy stderr error="..."
```

**Log Level Usage**:

- `debug`: Non-critical errors (stderr copy failure, resource close failure, etc.)
- `info`: Normal operation logs (server startup, shutdown, etc.)
- `warn`: Warnings (unused)
- `error`: Critical errors (process execution failure, request processing failure, etc.)

---

## Security Design

### Threat Model

**Threats Not Addressed (External Implementation Recommended)**:

- Authentication/Authorization
- TLS/HTTPS
- Rate limiting

**Addressed Threats**:

- Process timeout (DoS prevention)
- Resource exhaustion (timeout/Context management)
- Sensitive information leakage (log output control)

### Protection Mechanisms

**1. Timeout Control**:

Appropriate timeouts set for each operation:
- **ReadTimeout**: 30 seconds (HTTP request reading)
- **WriteTimeout**: 30 seconds (HTTP response writing)
- **ProcessTimeout**: 30 seconds (stdio process execution)
- **ShutdownTimeout**: 5 seconds (Graceful Shutdown)

**2. Context-based Cancellation**:

- Propagate HTTP request Context to process execution
- Terminate process on client disconnection (`exec.CommandContext`)

**3. Environment Variable Protection**:

- Pass sensitive information (tokens, etc.) via environment variables
- Don't include tokens in command-line arguments (process list exposure prevention)
- Don't output sensitive information in logs (except Debug level in structured logs)

**4. Process Isolation**:

- Launch independent process for each request
- No state sharing between processes
- Complete memory space separation

---

## Performance Design

### Concurrency

**HTTP Server**:

- Concurrent request processing automatically handled by Go goroutines
- Unlimited concurrent connections by default (external rate limiting recommended)

**Process Execution**:

- Independent process launch per request
- No mutual exclusion required between processes (stateless)

### Resource Management

**Memory**:

- Control memory usage through streaming processing
- Buffer size uses Go defaults (8192 bytes)

**Process**:

- Ensure process termination after request completion
- Proper cleanup on Context cancellation

**I/O**:

- Asynchronous stderr reading (goroutine)
- Prevent data race with `sync.WaitGroup`
- Prevent deadlock with WaitGroup.Wait() before `cmd.Wait()`

### Scaling

**Horizontal Scaling**:

- Easy due to stateless design
- Distribute to multiple instances with load balancer

**Vertical Scaling**:

- Concurrent processing count increases with CPU cores
- Memory usage proportional to number of launched processes

---

## Testing Design

### Testing Strategy

**Unit Testing (100% Coverage Target)**:

- `cmd/tumiki-mcp-http/main_test.go` - CLI flag parsing, configuration building
- `internal/proxy/server_test.go` - HTTP handler, header parsing
- `internal/process/executor_test.go` - Process execution, data race prevention

**Testing Approach**:

- Table-driven tests (structured multiple test cases)
- Coverage of normal cases, error cases, and edge cases
- HTTP testing with `httptest.NewRecorder`
- Testing with actual processes (echo, cat, sh, etc.)

### Test Coverage

**Coverage Exclusions (Integration Test Targets)**:

- `main()` function
- `startServer()` function (including signal handling)
- Server startup/shutdown processing

**Coverage Targets (100% Goal)**:

- All pure functions
- Header parsing logic (`parseHeaders`, `parseMapping`, `parseEnvVars`, etc.)
- Process execution logic (`Execute`, `envSlice`, etc.)
- Error handling (all error paths)

### Data Race Testing

Detect data races using Go's race detector.

Test execution command:
```bash
go test -race ./...
```

**Detected/Fixed Examples**:
- Concurrent access to `stderrBuf` → Synchronized with `sync.WaitGroup`

---

## Extensibility Design

### External Integration Points

**1. Reverse Proxy (Recommended)**:

- nginx, Caddy, Traefik, etc.
- Authentication/Authorization
- TLS/HTTPS
- Rate limiting
- Load balancing

**2. Monitoring/Logging**:

- Forward structured logs (JSON) to external systems
- Metrics collection (Prometheus, etc.)
- Health checks (periodic requests to /mcp)

**3. Orchestration**:

- Kubernetes deployment
- Multiple server management with Docker Compose
- Auto-start with Systemd

### Design Extensibility

**Supported by Current Design**:

- Multiple header mappings (unlimited)
- Multiple environment variables (unlimited)
- Any stdio process

**Features Recommended for External Implementation**:

- Authentication/Authorization (OAuth, JWT, etc.)
- TLS/HTTPS (certificate management)
- Rate limiting (Redis, etc.)
- Multi-backend load balancing
- Caching

### Simple Design Philosophy

**Principles**:

- Do one thing well (Unix philosophy)
- Delegate complex features externally
- Small, maintainable codebase

**Design Trade-offs**:

**Chosen**:

- Simplicity > Feature-rich
- External integration > Internal implementation
- Standard library > External dependencies

**Not Chosen**:

- Authentication features (implement with external reverse proxy)
- Configuration files (CLI flags only)
- Multi-server support (dedicated to single server)
- Health check endpoints (can use /mcp)

---

## Summary

### Architecture Characteristics

**Structure**:

- 2-package structure (proxy, process)
- Standard library only (no external dependencies)
- Small codebase (approximately 1000 lines)

**Quality**:

- 100% test coverage (for testable functions)
- Data race prevention (race detector compliant)
- Full golangci-lint compliance (errcheck, gocritic, etc.)

**Maintainability**:

- Easy to understand (small, simple design)
- Easy to debug (structured logging, error handling)
- Easy to extend (clear external integration points)
- Easy to replace (single responsibility principle)

This design realizes a lightweight, high-performance, and highly maintainable HTTP proxy.
