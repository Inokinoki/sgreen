# sgreen

A simplified screen-like terminal multiplexer written in pure Go, compatible with the `screen` command interface.

## Features

- Pure Go implementation (no CGO dependencies, no libc dependency)
- Screen-compatible commands: `new`, `attach`, `list`
- Detach with `Ctrl+A, d` (screen-compatible escape sequence)
- Session persistence across terminal sessions
- Cross-compilation support for:
  - Linux (amd64, arm64, armv7)
  - Windows (amd64, arm64)
  - macOS (amd64, arm64)
  - FreeBSD (amd64, arm64)
  - Android (arm64 only, amd64 requires CGO)

## Requirements

- Go 1.21 or later
- Make (optional, for using Makefile)

## Building

### Build for current platform

```bash
make build
# or
CGO_ENABLED=0 go build -o build/sgreen ./cmd/sgreen
```

### Build for all platforms

```bash
make all
```

This will create binaries in the `build/` directory for all supported platforms.

### Build for specific platform

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o build/sgreen-linux-amd64 ./cmd/sgreen

# Windows amd64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o build/sgreen-windows-amd64.exe ./cmd/sgreen

# macOS arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o build/sgreen-darwin-arm64 ./cmd/sgreen

# Android arm64
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags="-w -s" -o build/sgreen-android-arm64 ./cmd/sgreen

# FreeBSD amd64
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags="-w -s" -o build/sgreen-freebsd-amd64 ./cmd/sgreen
```

## Usage

### Create a new session

```bash
sgreen new --id mysession -- /bin/bash
```

### Attach to a session

```bash
sgreen attach --id mysession
```

Press `Ctrl+A, d` to detach from the session.

### List all sessions

```bash
sgreen list
```

### Running

```bash
make run
# or
go run ./cmd/sgreen
```

## Development

### Format code
```bash
make fmt
```

### Run tests
```bash
make test
```

### Clean build artifacts
```bash
make clean
```

## Notes

- All builds use `CGO_ENABLED=0` to ensure no C dependencies and static linking
- Binaries are stripped (`-ldflags="-w -s"`) to reduce size
- Uses `github.com/creack/pty` and `golang.org/x/term` which support pure Go syscalls (no CGO needed)
- Sessions are stored in `~/.sgreen/sessions/`
- On macOS, binaries may still link to system libraries (`libSystem.B.dylib`, `libresolv.9.dylib`) which are part of the OS and don't require external C libraries
- On Linux, binaries are fully static with no external dependencies

## License

MIT License - see LICENSE file for details.

