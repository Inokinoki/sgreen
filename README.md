# sgreen

A simplified screen-like terminal multiplexer written in pure Go, compatible with the `screen` command interface.

## Features

- Pure Go implementation (no CGO dependencies, no libc dependency)
- GNU screen-compatible command-line interface (`-r`, `-S`, `-ls`, `-d`)
- Detach with `Ctrl+A, d` (screen-compatible escape sequence)
- Session persistence across terminal sessions
- Cross-process session reattachment (reattach to sessions created in other terminals)
- Cross-compilation support for:
  - Linux (amd64, arm64, armv7)
  - Windows (amd64, arm64)
  - macOS (amd64, arm64)
  - FreeBSD (amd64, arm64)
  - Android (arm64 only, amd64 requires CGO)

## Requirements

- Go 1.24 or later
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

sgreen is compatible with GNU screen's command-line interface.

### Create a new session

```bash
# Create a new session (default command is /bin/sh)
sgreen

# Create a new session with a specific command
sgreen /bin/bash

# Create a named session
sgreen -S mysession /bin/bash
```

### Attach to a session

```bash
# Reattach to a detached session (auto-selects if only one)
sgreen -r

# Reattach to a specific session
sgreen -r mysession
```

### List all sessions

```bash
sgreen -ls
# or
sgreen -list
```

### Detach a session

```bash
# Detach a session (from within the session, press Ctrl+A, d)
# Or from command line:
sgreen -d [session]
```

Press `Ctrl+A, d` to detach from a session (screen-compatible).

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

## Versioning and Releases

- Versioning follows SemVer (`vMAJOR.MINOR.PATCH` tags).
- PR titles are validated with Conventional Commit prefixes (`feat:`, `fix:`, etc.).
- `Release Please` runs on `main` and opens/updates a release PR with the next semantic version and changelog updates.
- When a version tag like `v1.2.3` is pushed, the release workflow builds all platform binaries and publishes them as GitHub Release assets.
- Binary version output (`sgreen -v`) is injected at build time from the tag via linker flags.

## Notes

- All builds use `CGO_ENABLED=0` to ensure no C dependencies and static linking
- Binaries are stripped (`-ldflags="-w -s"`) to reduce size
- Uses `github.com/creack/pty` and `golang.org/x/term` which support pure Go syscalls (no CGO needed)
- Sessions are stored in `~/.sgreen/sessions/`
- On macOS, binaries may still link to system libraries (`libSystem.B.dylib`, `libresolv.9.dylib`) which are part of the OS and don't require external C libraries
- On Linux, binaries are fully static with no external dependencies

## License

MIT License - see LICENSE file for details.

