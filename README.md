# sgreen

A simplified, screen-like terminal multiplexer written in Go, with CLI behavior intentionally aligned with GNU `screen`.

## Features

- GNU screen-style CLI for common flows:
  - attach/reattach: `-r`, `-R`, `-RR`, `-x`
  - detach/power-detach: `-d`, `-D`, `-d -r`
  - naming/listing/maintenance: `-S`, `-ls`, `-list`, `-wipe`
  - command/control: `-X`, `-q`, `-m`
  - detached start parsing compatible with `-d -m` and `-dmS`
- Screen-like default auto session naming: `<pid>.<tty>.<host>`
- Detach key sequence: `Ctrl+A`, then `d`
- Session persistence and cross-process reattach
- Cross-compilation targets:
  - Linux (amd64, arm64, armv7)
  - Windows (amd64, arm64)
  - macOS (amd64, arm64)
  - FreeBSD (amd64, arm64)
  - Android (arm64)

## Requirements

- Go 1.24 or later
- Make (optional)

## Building

### Current platform

```bash
make build
# or
CGO_ENABLED=0 go build -o build/sgreen ./cmd/sgreen
```

### All targets

```bash
make all
```

## Usage

### Create sessions

```bash
# Default shell is $SHELL when set, otherwise /bin/sh
sgreen

# Named session
sgreen -S mysession /bin/bash

# Detached start (GNU screen style)
sgreen -dmS mysession /bin/sh -c 'sleep 60'
```

### Attach / Reattach

```bash
sgreen -r
sgreen -r mysession
sgreen -R mysession
sgreen -RR mysession
sgreen -x mysession
```

### List / Wipe

```bash
sgreen -ls
sgreen -list
sgreen -q -ls
sgreen -wipe
```

### Detach / Power-detach

```bash
# In session: Ctrl+A, d
sgreen -d mysession
sgreen -d -r mysession
sgreen -D mysession
```

### Send a command

```bash
sgreen -X quit -S mysession
```

### Help / Version

```bash
sgreen -help
sgreen -v
```

## Development

### Format

```bash
make fmt
# or
go fmt ./...
```

### Tests

```bash
make test
# or
go test ./...
```

### GNU screen behavior comparison

Requires a local `screen` binary (defaults to `/usr/bin/screen`).

```bash
go build -o build/sgreen ./cmd/sgreen
SGREEN_BIN="$(pwd)/build/sgreen" test/behavior/compare_with_gnu_screen.sh
```

Report output:

`test/behavior/gnu_screen_comparison_results.md`

## Notes

- Session files are stored under `~/.sgreen/sessions/`
- `SCREENDIR` is respected for screen-style socket/listing path display
- All CI checks include:
  - unit/behavior tests (`go test ./...`)
  - GNU-screen parity comparison on Linux/macOS when `screen` is available

## License

MIT License. See `LICENSE`.

