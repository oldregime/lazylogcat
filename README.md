# lazylogcat

A TUI for viewing Android logcat logs.

## IMPORTANT

This project is in early development. Expect bugs and missing features.

## Prerequisites

- Go 1.25.5 or higher
- `adb` (Android Debug Bridge) installed and in PATH

## Installation

```bash
go install github.com/parfenovvs/lazylogcat@latest
```

## Usage

```bash
# Launch the TUI
lazylogcat

# Enable debug logging
lazylogcat --debug
```

### Keyboard Shortcuts

**Device View:**
- `↑`/`k` or `↓`/`j` - Navigate devices
- `Enter` - Connect to selected device
- `r` - Refresh device list
- `Ctrl+c` - Quit

**Logcat View:**

*Navigation:*
- Arrow keys/mouse - Scroll through logs
- `G` - Jump to bottom

*Filtering & Display:*
- `Ctrl+f` - Open filter management (package, tag, text, log level, format options)
- `Ctrl+w` - Toggle soft wrap
- `Ctrl+l` - Cycle through log priority levels (V→D→I→W→E→F)

*Visual Mode & Copying:*
- `v` - Enter/exit visual mode
- `V` - Start/clear line selection in visual mode
- `j`/`↓` or `k`/`↑` - Navigate in visual mode
- `y` - Yank (copy) current line or selection to clipboard
- `Esc` - Exit visual mode or clear selection

*Connection:*
- `Ctrl+d` - Back to device selection
- `Ctrl+r` - Reconnect to logcat
