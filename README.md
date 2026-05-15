# cpssh

Copy a screenshot on your Mac. Paste it in Claude Code on your SSH server.

---

Claude Code running over SSH can't read your local clipboard. `cpssh` bridges the gap: it watches your Mac's clipboard for images, syncs them to your server over SSH, and puts a text reference in your clipboard so you can paste it straight into Claude Code.

```
[you copy a screenshot]
     ↓
cpssh syncs it → dev@myserver:~/.cpssh/img42.png
     ↓
clipboard now contains: [image:$HOME/.cpssh/img42.png]
     ↓
paste in Claude Code on SSH → Claude reads the file automatically
```

## Requirements

- macOS as your local machine (where you take screenshots)
- An SSH server reachable with a key (password auth not supported)
- [Claude Code](https://claude.ai/code) on the server

> **Linux local machine:** The daemon runs on Linux but on Linux it only puts the text reference in your clipboard — the original PNG is not preserved simultaneously. Full dual-clipboard support for Linux is planned for a future release.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/pravnyadv/cpssh/main/install.sh | bash
```

Then run setup:

```bash
cpssh setup
```

Setup will ask for your SSH server, pick an SSH key from `~/.ssh`, and install a background daemon that starts automatically on login.

**Using an AWS `.pem` key?** When setup shows the key picker, choose `[0] Enter path manually` and type the full path to your `.pem` file. Make sure the file has the right permissions first: `chmod 400 ~/path/to/key.pem`

## How to use

1. Take a screenshot with **Cmd+Shift+4** (saves to clipboard on macOS)
2. Paste in Claude Code on your SSH session — the sync happens in the background and the text reference is ready almost instantly
3. Claude reads the file and sees the image

You can attach multiple images to one prompt: each screenshot gets a unique filename (`img1.png`, `img2.png`, …). Copy → paste → copy → paste to stack them up.

The original PNG stays in your clipboard too, so pasting into Discord, Slack, or any image app still works normally.

## Commands

```
cpssh setup          First-time setup: add server, install daemon
cpssh status         Is the daemon running? Last sync time?
cpssh add-server     Add another SSH server to sync to
cpssh remove-server  Remove a server
cpssh pause          Stop syncing temporarily (daemon keeps running)
cpssh resume         Resume syncing
cpssh restart        Restart the daemon
cpssh logs           Tail the daemon log
cpssh uninstall      Remove daemon and config
```

## How it works

- A background daemon polls the clipboard every 300ms using the native macOS/Linux clipboard API
- On a new image, it generates a short filename (`imgN.png`) and pipes the raw bytes to the server over a single SSH call (no rsync, no scp — just `cat > file` via stdin)
- SSH ControlMaster reuses connections — first sync after daemon start costs ~1s for the handshake; subsequent syncs are near-instant
- After a successful sync, the daemon writes both the original PNG **and** a text reference `[image:$HOME/.cpssh/imgN.png]` to the clipboard simultaneously using NSPasteboard — terminal paste gets the text, image apps get the PNG
- The 10 most recent images are kept on the server; older ones are purged automatically on each sync
- Images larger than 500 KB are resampled before upload using `sips` (ships with macOS), keeping transfer fast

## Configuration

Config lives at `~/.config/cpssh/config.json`. You can edit it directly:

```json
{
  "servers": [
    {
      "host": "your.server.com",
      "user": "dev",
      "ssh_key": "/Users/you/.ssh/id_ed25519",
      "sync_path": "$HOME/.cpssh"
    }
  ],
  "settings": {
    "poll_interval_ms": 300,
    "max_file_size_kb": 2048,
    "compress_above_kb": 500,
    "keep_last_n_files": 10,
    "paused": false
  }
}
```

## Build from source

Requires Go and Xcode Command Line Tools on macOS (needed for the CGO clipboard write).

```bash
git clone https://github.com/pravnyadv/cpssh
cd cpssh
SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go build -o cpssh .
```

Note: `go install` will not work on macOS because CGO requires the SDK path to be set explicitly. Use the install script for a regular install, or the build command above when developing.

## License

MIT
