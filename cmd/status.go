package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/pravnyadv/cpssh/internal/daemon"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status and configured servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		pidPath, _ := config.PIDPath()
		pid, running := readPID(pidPath)
		switch {
		case running:
			fmt.Printf("Daemon: running (pid %d)\n", pid)
		case daemon.DaemonInstalled():
			fmt.Println("Daemon: installed but not running")
		default:
			fmt.Println("Daemon: not installed (run: cpssh setup)")
		}

		if cfg.Settings.Paused {
			fmt.Println("Status: paused")
		} else {
			fmt.Println("Status: active")
		}

		fmt.Printf("\nServers (%d):\n", len(cfg.Servers))
		for _, s := range cfg.Servers {
			fmt.Printf("  %s@%s → %s\n", s.User, s.Host, s.SyncPath)
		}

		logPath, _ := config.LogPath()
		if t := lastSyncTime(logPath); !t.IsZero() {
			fmt.Printf("\nLast sync: %s\n", t.Format(time.RFC822))
		}

		return nil
	},
}

func readPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	if err := proc.Signal(os.Signal(nil)); err != nil {
		return 0, false
	}
	return pid, true
}

func lastSyncTime(logPath string) time.Time {
	f, err := os.Open(logPath)
	if err != nil {
		return time.Time{}
	}
	defer f.Close()

	// Read only the last 8KB — log grows indefinitely so avoid full file load.
	const tailSize = 8 * 1024
	if info, err := f.Stat(); err == nil {
		if offset := info.Size() - tailSize; offset > 0 {
			f.Seek(offset, io.SeekStart)
		}
	}

	data, err := io.ReadAll(f)
	if err != nil {
		return time.Time{}
	}

	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "sync:") && strings.Contains(lines[i], "ok") {
			parts := strings.SplitN(lines[i], " ", 3)
			if len(parts) >= 2 {
				t, err := time.Parse("2006/01/02 15:04:05", parts[0]+" "+parts[1])
				if err == nil {
					return t
				}
			}
		}
	}
	return time.Time{}
}
