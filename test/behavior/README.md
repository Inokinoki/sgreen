# sgreen behavior tests

This document describes behavior tests for the sgreen CLI so that the same expectations can be checked across platforms (Linux, macOS, Windows, FreeBSD, etc.).

## Test environment

- **Session isolation**: Run CLI tests with a dedicated home directory (e.g. `HOME=$(mktemp -d)`) so that `~/.sgreen/sessions/` does not mix with the user’s real sessions.
- **Platforms**: Run the same test matrix on each supported OS (linux, darwin, windows, freebsd) where applicable; some tests are Unix-only or Windows-only as noted.
- **TTY**: Tests that start or attach to a session require a PTY. In CI, either use a PTY helper (e.g. `script`, `expect`, or Go’s `github.com/creack/pty`) or mark those as manual.

---

## 1. Non-interactive CLI (no session, no TTY)

These can run in any environment (pipes, CI, no TTY).

| ID | Description | Command | Expected exit code | Expected stdout/stderr | Platform |
|----|-------------|---------|--------------------|------------------------|----------|
| B1.1 | Version | `sgreen -v` | 0 | stdout contains "sgreen version 0.1.0" (or current version) | all |
| B1.2 | Help short | `sgreen -h` | 0 | stdout contains "Usage:" and "sgreen" | all |
| B1.3 | Help long | `sgreen -help` | 0 | same as B1.2 | all |
| B1.4 | List when no sessions | `sgreen -ls` | 0 | stdout contains "No Sockets found" (or "No screen session") | all |
| B1.5 | List alternative flag | `sgreen -list` | 0 | same behavior as B1.4 | all |
| B1.6 | Reattach when no sessions | `sgreen -r` | non-zero | stderr contains "No screen session found" | all |
| B1.7 | Reattach to missing name | `sgreen -r nosuchsession123` | non-zero | stderr contains "No screen session found" and "nosuchsession123" | all |
| B1.8 | Wipe when no sessions | `sgreen -wipe` | 0 | stdout contains "No dead sessions found" | all |
| B1.9 | Detach when no sessions | `sgreen -d` | non-zero | stderr contains "No screen session found" | all |
| B1.10 | Power detach when no sessions | `sgreen -D` | non-zero | stderr contains "No screen session found" | all |
| B1.11 | Send command when no sessions | `sgreen -X stuff x` | non-zero | stderr contains "No screen session found" | all |
| B1.12 | Unknown flag | `sgreen -unknown` | non-zero | stderr mentions flag or usage | all |

---

## 2. Session lifecycle (requires TTY / PTY)

These create or attach to sessions; run only when a PTY is available (or in a PTY harness).

| ID | Description | Steps | Expected | Platform |
|----|-------------|--------|----------|----------|
| B2.1 | Create named session | `sgreen -S test_session -e /bin/sh -c "exit 0"` (or platform shell) | exit 0; session appears in `sgreen -ls` briefly; session file under `~/.sgreen/sessions/` | Unix (PTY); Windows (conhost) |
| B2.2 | List shows one session | Start session in background (PTY), then `sgreen -ls` | One line with session name and (Attached) or (Detached) | all |
| B2.3 | Reattach to named session | Create detached session, then `sgreen -r test_session` | Attaches; same session | all |
| B2.4 | -R reattach or create | No sessions: `sgreen -R -S rtest` → creates. Then `sgreen -R -S rtest` → reattaches | First run creates, second reattaches | all |
| B2.5 | -RR force reattach | Session attached elsewhere; `sgreen -RR -S name` | Force detach elsewhere and attach here | Unix (PTY); Windows |
| B2.6 | -D power detach then attach | Attached session; `sgreen -D -S name` from another process | Session detached and attach succeeds | all |
| B2.7 | -d detach (from command line) | Attached session; `sgreen -d -S name` | Session becomes detached (or error if no attached session) | all |
| B2.8 | -x multiuser attach | Session exists; `sgreen -x -S name` | Attach without detaching (multiuser) | all |
| B2.9 | -wipe removes dead session | Create session, kill shell only, then `sgreen -wipe` | "Removed 1 dead session(s)" and session no longer in list | all |
| B2.10 | -X send command | Attached or detached session; `sgreen -X stuff "echo hi" -S name` | Command executed in session (verify output if possible) | all |

---

## 3. Flags and config (non-interactive where possible)

| ID | Description | Command | Expected | Platform |
|----|-------------|---------|----------|----------|
| B3.1 | -q quiet | `sgreen -q -ls` | exit 0; no extra startup messages | all |
| B3.2 | -m ignore STY | With `STY=123.pts.host` set, `sgreen -m -ls` | Uses normal behavior; does not try to attach from STY | all |
| B3.3 | -S name | `sgreen -S myname -ls` then create session with `-S myname` | Session name is "myname" in list | all |
| B3.4 | -c config file | `sgreen -c /nonexistent -ls` | exit 0 (config optional); if config missing, may warn | all |
| B3.5 | -v version format | `sgreen -v` | Exactly 3 lines: version, description, "GNU screen" compatibility line | all |

---

## 4. Platform-specific behavior

| ID | Description | Condition | Expected | Platform |
|----|-------------|-----------|----------|----------|
| B4.1 | Session dir | Any | Sessions stored under `$HOME/.sgreen/sessions/*.json` | all |
| B4.2 | List format | `sgreen -ls` with one session | Line like `\tPID.TTY.HOST\t(Attached\|Detached)\tDATE TIME\t(NAME)` | all |
| B4.3 | PTY path persistence | Unix: create session, detach, reattach | Reconnect via PtsPath works (no "no active PTY" when process alive) | Linux, macOS, FreeBSD |
| B4.4 | Windows no PTY path | Windows: session metadata | PtsPath may be empty; reattach still works via process/console | Windows |
| B4.5 | Default shell | No `-s`, no `SHELL` | Unix: `/bin/sh` or similar; Windows: `cmd` or `%COMSPEC%` | per OS |
| B4.6 | Help / status line | UI text | On Windows, messages that reference "Ctrl+A" may show alternative (e.g. "C-a") | Windows (ui/help.go, status.go) |
| B4.7 | Detach keeper | Unix: detach from session | SGREEN_DETACH_KEEPER=1 child holds PTY master so subprocess does not get SIGHUP | Unix only |
| B4.8 | Reconnect PTY | Unix: kill sgreen, leave shell running; new sgreen -r | Reconnect to same PTY if path stored and process alive | Linux, macOS, FreeBSD |

---

## 5. Error messages and exit codes

| ID | Description | Command | Expected exit code | Expected stderr |
|----|-------------|---------|--------------------|-----------------|
| B5.1 | Reattach no session | `sgreen -r` | 1 | "No screen session found." |
| B5.2 | Reattach wrong name | `sgreen -r wrongname` | 1 | "No screen session found: wrongname" |
| B5.3 | Multiple sessions, -r | Two sessions, `sgreen -r` (no name) | 1 | "There are several detached sessions" or similar; list printed |
| B5.4 | -d no attached session | No sessions or only detached: `sgreen -d` | 1 | "No attached screen session found" or "No screen session found" |
| B5.5 | -X no session | `sgreen -X stuff x` | 1 | "No screen session found." |
| B5.6 | Permission denied | Multiuser session with attach denied | 1 | "Permission denied" and session/user info | when ACLs implemented |

---

## 6. Running the tests

- **Automated (no TTY)**  
  Run the B1.* and B3.* (and B5.* where applicable) cases via:
  ```bash
  go test -v ./test/behavior/...
  ```
  Session state is isolated by a temp `HOME`. For faster runs, build the binary once and point to it:
  ```bash
  make build
  SGREEN_BINARY=./build/sgreen go test -v ./test/behavior/...
  ```
  See `cli_test.go` in this directory.

- **CI**  
  Add a job (or matrix) that builds the binary for each OS/arch, then runs the same CLI behavior test suite with an isolated `HOME`. Skip or mark TTY-only tests when not running under a PTY.

- **Manual / TTY**  
  For B2.* and B4.*, run the steps by hand (or with a PTY test harness) on each platform and compare to the table.

- **Platform matrix**  
  - Linux (amd64, arm64): all tests.  
  - macOS (amd64, arm64): all tests; PTY behavior as on Unix.  
  - Windows (amd64, arm64): all tests; B4.4, B4.6 apply; B4.7 N/A.  
  - FreeBSD: same as Linux/macOS for Unix-specific rows.  
  - Android: run non-interactive and session tests if PTY available; reconnect behavior may differ.

---

## Summary

- **Non-interactive (B1, B3, B5)**: Safe to run in CI on all platforms with a temp HOME and no TTY.
- **Session lifecycle (B2)**: Requires PTY (or manual run) on each platform.
- **Platform-specific (B4)**: Validate once per OS (and arch if behavior differs).

Use this matrix to add or extend automated tests in `cli_test.go` and to document any platform differences as they are discovered.
