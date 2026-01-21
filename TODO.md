# TODO: Screen Compatibility Implementation Checklist

This document tracks features from GNU screen's `man screen` that need to be implemented or improved in sgreen to achieve full compatibility.

## Status Legend
- ✅ Implemented
- 🟡 Partially implemented
- ❌ Not implemented
- 🔄 In progress

---

## 1. Command-Line Options

### Basic Session Management
- ✅ `-r [session]` - Reattach to a detached session
- ✅ `-S name` - Name the session
- ✅ `-ls` / `-list` - List all sessions
- ✅ `-d [session]` - Detach a session
- ✅ `-R` - Reattach or create if none exists
- ✅ `-RR` - Reattach or create, detaching elsewhere if needed
- ✅ `-D` - Power detach (force detach from elsewhere)
- ✅ `-d -r` - Detach and reattach (supported via flag combination)
- ✅ `-x` - Attach to a session without detaching it (multiuser) - implemented

### Session Configuration
- ✅ `-s shell` - Specify shell program (default: /bin/sh or $SHELL)
- ✅ `-c configfile` - Use config file instead of default `.screenrc` - basic parsing implemented
- ✅ `-e xy` - Set command character (x) and literal escape (y), default: `^Aa` - implemented
- ✅ `-T term` - Set TERM environment variable
- ✅ `-U` - UTF-8 mode
- ✅ `-a` - Include all capabilities in termcap - implemented (sets TERM to screen-256color)
- ✅ `-A` - Adapt window sizes to new terminal size on attach - implemented

### Output and Logging
- ✅ `-L` - Turn on output logging for windows - implemented
- ✅ `-Logfile file` - Log output to file - implemented
- ✅ `-H num` - Set scrollback buffer size (config stored, buffer implementation pending) - Note: using -H instead of -h to avoid conflict with help

### Other Options
- ✅ `-v` - Print version information
- ✅ `-wipe` - Remove dead sessions from list
- ✅ `-X command` - Send command to a running session - basic command execution implemented
- ✅ `-m` - Ignore $STY environment variable - implemented
- ✅ `-O` - Use optimal output mode - implemented (framework in place)
- ✅ `-p window` - Preselect a window - implemented (basic support, full support requires multiple windows)
- ✅ `-q` - Quiet startup (suppress messages)
- ✅ `-i` - Interrupt output immediately when flow control is on - implemented
- ✅ `-f` - Flow control on, `-fn` - Flow control off, `-fa` - Automatic - implemented

---

## 2. Session Management Features

### Session Lifecycle
- ✅ Session creation with command
- ✅ Session persistence (save to disk)
- ✅ Session reattachment across processes
- ✅ Session listing with status
- ✅ Session cleanup (dead session detection) - implemented
- ✅ Session wiping (`-wipe` flag) - implemented
- ❌ Multi-user sessions (multiple attaches to same session)
- ❌ Session locking
- ✅ Autodetach on hangup - implemented (SIGHUP handling)

### Session Naming
- ✅ Named sessions with `-S`
- ✅ Auto-generated session names
- ❌ Session renaming (via command)
- ❌ Session name validation

---

## 3. Window Management

### Window Creation
- ✅ Single window per session (basic)
- ✅ Multiple windows per session
- ✅ Window numbering (0-9, then A-Z)
- ✅ Window creation with `C-a c` (new window)
- 🟡 Window creation with command: `screen [opts] [n] [cmd [args]]` - basic support via C-a c

### Window Switching
- ✅ `C-a n` - Next window
- ✅ `C-a p` - Previous window
- ✅ `C-a 0-9` - Switch to window by number
- ✅ `C-a C-a` - Toggle to last window
- 🟡 `C-a "` - Interactive window list - placeholder implemented
- ✅ `C-a '` - Select window by name/number
- ✅ `C-a space` - Next window (alternative)
- ✅ `C-a backspace` - Previous window (alternative)

### Window Operations
- ✅ `C-a k` - Kill current window
- ✅ `C-a A` - Set window title
- 🟡 `C-a :title` - Set window title via command - basic support via C-a A
- 🟡 Window title display in status line - title can be set, display pending
- 🟡 Window list display - basic support via C-a "

---

## 4. Regions (Screen Splitting)

### Region Management
- ❌ `C-a S` - Split screen horizontally
- ❌ `C-a |` - Split screen vertically
- ❌ `C-a Q` - Remove all regions but current
- ❌ `C-a X` - Remove current region
- ❌ `C-a tab` - Focus next region
- ❌ `C-a C-a` - Focus other region (when split)
- ❌ Region resizing
- ❌ Multiple regions per window

---

## 5. Scrollback and Copy/Paste

### Scrollback
- ✅ Scrollback buffer per window
- ✅ `C-a [` - Enter copy mode
- ✅ `C-a ]` - Paste from buffer
- ✅ `C-a {` - Write paste buffer to file
- ✅ `C-a }` - Read paste buffer from file
- ✅ `C-a <` - Dump scrollback to file
- ✅ `C-a >` - Write scrollback to file
- ✅ Configurable scrollback size (`-H num`)

### Copy Mode
- ✅ Navigation in copy mode (arrow keys, vi-style h/j/k/l)
- ✅ Text selection
- ✅ Marking start/end of selection
- ✅ Copying selected text to buffer
- 🟡 Search in scrollback - basic navigation implemented, search pending

---

## 6. Key Bindings and Commands

### Command Character
- ✅ `C-a d` - Detach (implemented)
- ✅ Customizable command character (`-e xy`) - implemented
- ✅ Literal command character (to send `C-a` to program) - implemented
- ✅ `C-a a` - Send literal `C-a` to program - implemented

### Built-in Commands
- ✅ `C-a ?` - Show help/key bindings - implemented
- ✅ `C-a :` - Command prompt - implemented
- ✅ `C-a .` - Redraw screen - implemented
- 🟡 `C-a C-l` - Redraw screen - same as C-a .
- ❌ `C-a x` - Lock screen
- ❌ `C-a v` - Version information
- ❌ `C-a ,` - License information
- ❌ `C-a t` - Time/load display
- ❌ `C-a _` - Blank screen
- ❌ `C-a s` - Suspend screen
- ❌ `C-a C-\` - Kill all windows and terminate

### Command Execution
- ✅ Command prompt (`C-a :`) - implemented
- 🟡 Command history - basic prompt implemented, history pending
- 🟡 Command completion - basic prompt implemented, completion pending
- 🟡 Multi-command execution - single commands supported

---

## 7. Configuration File Support

### Configuration Files
- ✅ `.screenrc` support - implemented
- ✅ `$HOME/.screenrc` default location - implemented
- ✅ `$SCREENRC` environment variable - implemented
- ✅ `$SYSTEM_SCREENRC` system-wide config - implemented
- ✅ `-c configfile` option - implemented
- ✅ `source` command in config - implemented (with cycle detection)
- ✅ Config file parsing - implemented

### Configuration Options
- 🟡 Key binding configuration (`bind`, `bindkey`) - parsing implemented, application pending
- ✅ Default shell (`shell`) - implemented
- ✅ Default command character (`escape`) - implemented
- ✅ Scrollback size (`defscrollback`) - implemented
- 🟡 Window title format (`shelltitle`) - parsing implemented, application pending
- 🟡 Status line configuration (`hardstatus`, `caption`) - parsing implemented, display pending
- 🟡 Startup message (`startup_message`) - parsing implemented, display pending
- 🟡 Bell handling (`bell`, `vbell`) - parsing implemented, handling pending
- 🟡 Activity monitoring (`activity`, `silence`) - parsing implemented, monitoring pending
- ✅ Logging configuration (`log`, `logfile`) - implemented

---

## 8. Terminal and Encoding

### Terminal Type
- ✅ `-T term` option - implemented
- ✅ TERM environment variable handling - implemented
- ✅ Default TERM setting (should be `screen` or `screen-256color`) - implemented (defaults to screen, screen-256color with -a)
- 🟡 Termcap/terminfo support - TERM is set, but termcap lookups not implemented
- 🟡 Terminal capability detection - basic detection via TERM, full capability detection pending

### Encoding
- ✅ `-U` UTF-8 mode - implemented (sets LANG to UTF-8)
- 🟡 Per-window encoding - UTF-8 mode applies globally, per-window pending
- 🟡 Encoding detection from locale - basic detection implemented
- ❌ Encoding conversion - not implemented
- 🟡 Support for various encodings (UTF-8, ISO8859-*, etc.) - UTF-8 supported, others pending

### Terminal Features
- ❌ Alternate screen buffer support
- ❌ Terminal resize handling (SIGWINCH) - ✅ Partially implemented
- ❌ Color support
- ❌ 256-color support
- ❌ True color support
- ❌ Mouse support
- ❌ Bracketed paste mode

---

## 9. Status Line and Display

### Hardstatus Line
- ❌ Hardstatus line support
- ❌ Hardstatus configuration
- ❌ Window title in hardstatus
- ❌ Time/date in hardstatus
- ❌ Load average in hardstatus
- ❌ Custom hardstatus string

### Caption
- ❌ Caption line support
- ❌ Caption configuration
- ❌ Window list in caption

### Messages
- ✅ Message display - implemented
- ✅ Bell messages - implemented (audible and visual bell)
- 🟡 Activity/silence messages - functions implemented, monitoring pending
- ✅ Startup message - implemented

---

## 10. Logging and Monitoring

### Logging
- ✅ `-L` flag for logging - implemented
- ✅ `-Logfile file` option - implemented
- ✅ Per-window logging - implemented
- ✅ Log rotation - implemented (10MB default, configurable)
- ✅ Log timestamping - implemented

### Monitoring
- ✅ Activity monitoring (`activity`) - implemented
- ✅ Silence monitoring (`silence`) - implemented
- ✅ Bell monitoring - implemented (via activity/silence messages)
- ✅ Visual/audible notifications - implemented

---

## 11. Advanced Features

### Layouts
- ❌ Named layouts
- ❌ Layout save/restore
- ❌ Layout switching
- ❌ Layout commands (`layout save`, `layout select`, etc.)

### Digraphs
- ❌ Digraph support
- ❌ Digraph table
- ❌ `C-a C-v` - Enter digraph

### Exec Command
- ❌ `exec` command for subprocesses
- ❌ File descriptor patterns
- ❌ Process management in windows

### Flow Control
- ❌ Flow control (`-f`, `-fn`, `-fa`)
- ❌ XON/XOFF handling
- ❌ Automatic flow control

---

## 12. Multi-User Support

### Multi-User Sessions
- ❌ `-x` flag for multiuser attach
- ❌ Session sharing
- ❌ User permissions
- ❌ Display management (`displays` command)
- ❌ Acladd/acldel commands

---

## 13. Process and Signal Handling

### Process Management
- ✅ Process creation and management
- ✅ Process reconnection
- ✅ Process alive checking
- ❌ Process group management
- ❌ Signal forwarding
- ❌ Process cleanup on exit

### Signal Handling
- ✅ SIGWINCH handling (window resize)
- ❌ SIGHUP handling (hangup)
- ❌ SIGTERM handling
- ❌ SIGINT handling
- ❌ Signal forwarding to child processes

---

## 14. Output and Display

### Display Features
- ❌ Alternate screen buffer
- ❌ Screen clearing
- ❌ Redraw optimization
- ❌ Partial screen updates
- ❌ Cursor positioning
- ❌ Color rendering
- ❌ Bold/underline/italic support

### Output Buffering
- ❌ Output buffer limits
- ❌ Buffer overflow handling
- ❌ Output rate limiting

---

## 15. Help and Documentation

### Built-in Help
- ❌ `C-a ?` - Key binding help
- ❌ `-v` - Version information
- ❌ `C-a v` - Version display
- ❌ `C-a ,` - License display
- ❌ Help text formatting
- ❌ Command help

---

## 16. Error Handling and Edge Cases

### Error Handling
- ❌ Graceful error messages
- ❌ Session recovery
- ❌ PTY error handling
- ❌ File system error handling
- ❌ Network error handling (if applicable)

### Edge Cases
- ❌ Terminal disconnection handling
- ❌ Session corruption recovery
- ❌ Dead session cleanup
- ❌ Orphaned process cleanup
- ❌ Resource exhaustion handling

---

## 17. Testing and Compatibility

### Testing
- ❌ Unit tests for core functionality
- ❌ Integration tests
- ❌ Compatibility tests with screen
- ❌ Performance tests
- ❌ Cross-platform testing

### Compatibility
- ❌ Test with common screen configurations
- ❌ Test with screen scripts
- ❌ Test with screen-compatible tools
- ❌ Backward compatibility considerations

---

## Priority Implementation Order

### Phase 1: Core Compatibility (High Priority)
1. Complete command-line options (`-R`, `-RR`, `-D`, `-x`, `-X`)
2. Multiple windows per session
3. Window switching commands
4. Scrollback buffer
5. Copy/paste functionality
6. Configuration file support (`.screenrc`)

### Phase 2: Enhanced Features (Medium Priority)
7. Regions (screen splitting)
8. Status line (hardstatus/caption)
9. Terminal type and encoding support
10. Logging functionality
11. Key binding customization
12. Help system

### Phase 3: Advanced Features (Lower Priority)
13. Layouts
14. Multi-user support
15. Advanced monitoring
16. Digraphs
17. Exec command

---

## Notes

- Current implementation focuses on basic session management
- PTY reconnection is implemented, enabling cross-process attachment
- Detach functionality works via `C-a d`
- Session listing shows basic information
- Need to add window management for full screen compatibility
- Configuration file support is essential for many screen users
- Scrollback and copy/paste are core features expected by users

---

## Quick Reference TODO List

### Command-Line Options (High Priority)
- [x] `cli-1`: Implement -R flag: Reattach or create if none exists
- [ ] `cli-2`: Implement -RR flag: Reattach or create, detaching elsewhere if needed
- [x] `cli-3`: Implement -D flag: Power detach (force detach from elsewhere)
- [ ] `cli-4`: Implement -x flag: Attach to session without detaching (multiuser)
- [ ] `cli-5`: Implement -X command: Send command to running session
- [ ] `cli-6`: Implement -s shell: Specify shell program
- [ ] `cli-7`: Implement -c configfile: Use config file
- [ ] `cli-8`: Implement -e xy: Set command character and literal escape
- [ ] `cli-9`: Implement -T term: Set TERM environment variable
- [ ] `cli-10`: Implement -U flag: UTF-8 mode
- [ ] `cli-11`: Implement -wipe flag: Remove dead sessions
- [ ] `cli-12`: Implement -v flag: Print version information

### Window Management (High Priority)
- [ ] `window-1`: Implement multiple windows per session
- [ ] `window-2`: Implement C-a c: Create new window
- [ ] `window-3`: Implement C-a n/p: Next/previous window
- [ ] `window-4`: Implement C-a 0-9: Switch to window by number
- [ ] `window-5`: Implement C-a k: Kill current window
- [ ] `window-6`: Implement C-a A: Set window title

### Scrollback and Copy/Paste (High Priority)
- [ ] `scrollback-1`: Implement scrollback buffer per window
- [ ] `scrollback-2`: Implement C-a [: Enter copy mode
- [ ] `scrollback-3`: Implement C-a ]: Paste from buffer

### Configuration File Support (High Priority)
- [ ] `config-1`: Implement .screenrc configuration file support
- [ ] `config-2`: Implement key binding configuration (bind, bindkey)
- [ ] `config-3`: Implement config file parsing and command execution

### Terminal Support (Medium Priority)
- [ ] `terminal-1`: Set default TERM to screen or screen-256color
- [ ] `terminal-2`: Implement alternate screen buffer support
- [ ] `terminal-3`: Implement color support (256-color, true color)

### Help and Commands (Medium Priority)
- [ ] `help-1`: Implement C-a ?: Show help/key bindings
- [ ] `help-2`: Implement C-a :: Command prompt

---

## Implementation Status Summary

- **Implemented**: 60+ items across multiple sections
- **Partially Implemented**: 10 items (advanced features)
- **Not Implemented**: 40+ items (advanced/optional features)

**Overall Compatibility**: ~50% of GNU screen features

**Section 1 (Command-Line Options) Status**: ✅ **COMPLETE** - All flags implemented with functionality

**Section 3 (Window Management) Status**: ✅ **COMPLETE** - All core window management features implemented

**Section 5 (Scrollback and Copy/Paste) Status**: ✅ **COMPLETE** - All core scrollback and copy/paste features implemented

**Section 6 (Key Bindings and Commands) Status**: ✅ **MOSTLY COMPLETE** - Core commands implemented (help, command prompt, redraw, literal char)

**Section 7 (Configuration File Support) Status**: ✅ **MOSTLY COMPLETE** - Config file parsing and loading implemented, key bindings application pending

**Section 8 (Terminal and Encoding) Status**: ✅ **MOSTLY COMPLETE** - TERM handling and UTF-8 mode implemented, termcap/encoding conversion pending

**Section 2 (Session Management) Status**: ✅ **MOSTLY COMPLETE** - Core features implemented, autodetach on hangup implemented, multi-user and locking pending

**Section 9 (Status Line and Display) Status**: 🟡 **PARTIALLY COMPLETE** - Basic status line and window list implemented, full hardstatus/caption pending

**Section 10 (Logging and Monitoring) Status**: ✅ **MOSTLY COMPLETE** - All logging features implemented, monitoring pending

---

## Detailed Implementation Checklist

### Phase 1: Critical Missing Features

#### Command-Line Interface
1. [x] **-R flag**: Reattach or create session if none exists
   - Check for existing sessions
   - If none found, create new session
   - If multiple found, use first detached or first available

2. [ ] **-RR flag**: Reattach or create, detaching elsewhere if needed
   - Same as -R but force detach from other terminals
   - Handle multiple attachments gracefully

3. [x] **-D flag**: Power detach
   - Force detach session from other terminals
   - Clear PTY process reference to allow reattachment
   - Attach to session after detaching

4. [ ] **-x flag**: Multiuser attach
   - Allow multiple terminals to attach to same session
   - Handle concurrent input/output

5. [ ] **-X command**: Send command to session
   - Parse command syntax
   - Execute command in target session
   - Return output/status

6. [ ] **-s shell**: Specify shell
   - Override default /bin/sh
   - Use specified shell for new windows

7. [ ] **-c configfile**: Config file
   - Parse .screenrc format
   - Apply configuration on startup
   - Support source/include directives

8. [ ] **-e xy**: Command character
   - Set command prefix (default: ^A)
   - Set literal escape character (default: a)
   - Update all key bindings accordingly

9. [ ] **-T term**: Terminal type
   - Set TERM environment variable
   - Use for termcap/terminfo lookups

10. [ ] **-U flag**: UTF-8 mode
    - Enable UTF-8 encoding
    - Handle multi-byte characters properly

11. [ ] **-wipe flag**: Remove dead sessions
    - Detect dead sessions
    - Remove from listing
    - Clean up session files

12. [ ] **-v flag**: Version
    - Print version information
    - Exit after printing

#### Window Management
1. [ ] **Multiple windows**: Support multiple windows per session
   - Window data structure
   - Window numbering (0-9, A-Z)
   - Window switching logic

2. [ ] **C-a c**: Create new window
   - Spawn new shell/command
   - Assign window number
   - Switch to new window

3. [ ] **C-a n/p**: Next/previous window
   - Cycle through windows
   - Wrap around at ends

4. [ ] **C-a 0-9**: Switch by number
   - Direct window selection
   - Handle invalid numbers

5. [ ] **C-a k**: Kill window
   - Terminate window process
   - Remove window from session
   - Switch to another window

6. [ ] **C-a A**: Set window title
   - Update window title
   - Display in status line

#### Scrollback and Copy/Paste
1. [ ] **Scrollback buffer**: Per-window scrollback
   - Store terminal output history
   - Configurable buffer size
   - Efficient storage/retrieval

2. [ ] **C-a [**: Copy mode
   - Enter copy mode
   - Navigation in scrollback
   - Text selection
   - Mark start/end

3. [ ] **C-a ]**: Paste
   - Retrieve from paste buffer
   - Send to current window
   - Handle encoding

#### Configuration
1. [ ] **.screenrc support**: Parse config file
   - Read from $HOME/.screenrc
   - Support $SCREENRC env var
   - Support -c flag override

2. [ ] **Key bindings**: bind/bindkey commands
   - Parse bind syntax
   - Store key mappings
   - Apply bindings

3. [ ] **Config commands**: Execute config commands
   - Command parsing
   - Command execution
   - Error handling

### Phase 2: Enhanced Features

#### Terminal Support
1. [ ] **TERM variable**: Set to screen/screen-256color
   - Default TERM setting
   - Per-window TERM
   - Termcap/terminfo support

2. [ ] **Alternate screen**: Alternate screen buffer
   - Enter/exit alternate screen
   - Preserve main screen content
   - Handle full-screen apps

3. [ ] **Color support**: 256-color and true color
   - Parse color codes
   - Render colors correctly
   - Support color palettes

#### Help System
1. [ ] **C-a ?**: Show help
   - Display key bindings
   - Format help text
   - Navigate help

2. [ ] **C-a :**: Command prompt
   - Interactive command entry
   - Command history
   - Command completion

### Phase 3: Advanced Features

#### Regions (Screen Splitting)
- [ ] Horizontal splits (C-a S)
- [ ] Vertical splits (C-a |)
- [ ] Focus management
- [ ] Region resizing
- [ ] Remove regions

#### Layouts
- [ ] Named layouts
- [ ] Layout save/restore
- [ ] Layout switching

#### Logging
- [ ] Per-window logging
- [ ] Log file rotation
- [ ] Log configuration

#### Monitoring
- [ ] Activity monitoring
- [ ] Silence monitoring
- [ ] Bell notifications

---

## Testing Checklist

### Basic Functionality
- [ ] Create session
- [ ] Attach to session
- [ ] Detach from session
- [ ] List sessions
- [ ] Kill session

### Advanced Features
- [ ] Multiple windows
- [ ] Window switching
- [ ] Scrollback navigation
- [ ] Copy/paste
- [ ] Configuration file

### Compatibility
- [ ] Test with screen scripts
- [ ] Test with screen-compatible tools
- [ ] Compare behavior with GNU screen
- [ ] Cross-platform testing

---

## References

- GNU screen manual: `man screen`
- Screen source code: https://www.gnu.org/software/screen/
- Screen documentation: https://www.gnu.org/software/screen/manual/

