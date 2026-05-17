# cpssh

Copy a screenshot on your Mac. Paste it in Claude Code on your SSH server.

---

Claude Code running over SSH can't read your local clipboard. `cpssh` bridges the gap: it watches your Mac's clipboard for images, syncs them to your server over SSH, and adds a text reference alongside the image — terminal paste gives the path, image apps still get the original PNG.

```
[you copy a screenshot]
     ↓
cpssh syncs it → dev@myserver:~/.cpssh/img5.png
     ↓
clipboard now contains: original PNG + [~/.cpssh/img5.png]
     ↓
paste in Claude Code on SSH → terminal gets the path → Claude reads the file
paste in Slack/Preview        → still gets the original image
```

## Requirements

- macOS as your local machine (where you take screenshots)
- An SSH server reachable with a key (password auth not supported)
- [Claude Code](https://claude.ai/code) on the server
- `pngpaste` — `brew install pngpaste`

> **Linux local machine:** The daemon runs on Linux but only the text reference is written to the clipboard (the original image is replaced). macOS gets dual-type clipboard via NSPasteboard.

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

You can attach multiple images to one prompt: filenames cycle through `img1.png`–`img10.png`. Copy → paste → copy → paste to stack them up.

The original PNG stays in your clipboard alongside the text reference, so pasting into Discord, Slack, Preview, or any image app still works normally.

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

- A background daemon polls the clipboard every 300ms using `pngpaste`
- On a new image, it generates a cycling filename (`img1.png`–`img10.png`) and pipes the raw bytes to the server over a single SSH call (no rsync, no scp — just `cat > file` via stdin)
- SSH ControlMaster reuses connections — first sync after daemon start costs ~1s for the handshake; subsequent syncs are near-instant
- After a successful sync, the daemon writes both the original PNG **and** the text reference `[~/.cpssh/imgN.png]` to the clipboard simultaneously via NSPasteboard — terminal paste gets the text, image apps get the PNG
- Up to 10 images are kept on the server (filenames cycle through `img1.png`–`img10.png`)
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

Requires Go 1.21+ and Xcode Command Line Tools on macOS (CGo uses the Cocoa framework for the clipboard write).

```bash
git clone https://github.com/pravnyadv/cpssh
cd cpssh
SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go build -o cpssh .
```

`go install` won't work on macOS because CGo needs the SDK path. Use the install script for a regular install, or the build command above when developing.

## License

MIT
