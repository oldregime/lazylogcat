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

### Keyboard Shortcuts

**Device View:**
- Navigate devices with arrow keys or vim keys (`j`/`k`)
- `Enter` - Connect to selected device
- `r` - Refresh device list

**Logcat View:**

*Navigation:*
- Arrow keys/mouse - Scroll through logs
- `G` - Jump to bottom

*Filtering & Display:*
- `Alt+p` - Filter by package name
- `Alt+c` - Toggle color output
- `Alt+t` - Toggle tag format
- `Alt+w` - Toggle soft wrap
- `p` - Cycle through log priority levels (V→D→I→W→E→F)

*Visual Mode & Copying:*
- `v` - Enter/exit visual mode
- `V` - Start/clear line selection in visual mode
- `j`/`k` or arrow keys - Navigate in visual mode
- `y` - Yank (copy) current line or selection to clipboard
- `Esc` - Exit visual mode

*Connection:*
- `Ctrl+d` - Back to device selection
- `Ctrl+r` - Reconnect to logcat
