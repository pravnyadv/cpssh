package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/pravnyadv/cpssh/internal/daemon"
	"github.com/pravnyadv/cpssh/internal/server"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove daemon and config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !prompt("This will remove cpssh completely. Continue? [y/N]: ", false) {
			fmt.Println("Aborted.")
			return nil
		}

		// Load config before removing it — needed for server cleanup.
		cfg, cfgErr := config.Load()

		cleanServers := false
		if cfgErr == nil && len(cfg.Servers) > 0 {
			cleanServers = prompt("Also remove synced image directories from your server(s)? [y/N]: ", false)
		}

		if daemon.DaemonInstalled() {
			fmt.Println("Removing daemon...")
			if err := daemon.UninstallDaemon(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}

		cfgDir, err := config.ConfigDir()
		if err == nil {
			os.RemoveAll(cfgDir)
		}

		if cleanServers {
			for _, srv := range cfg.Servers {
				fmt.Printf("Cleaning up %s...\n", serverAddr(srv))
				sshArgs := append(server.BaseArgs(srv),
					fmt.Sprintf("%s@%s", srv.User, srv.Host),
					fmt.Sprintf("rm -rf %s", shellSingleQuote(srv.SyncPath)),
				)
				if err := exec.Command("ssh", sshArgs...).Run(); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not remove %s:%s — %v\n", srv.Host, srv.SyncPath, err)
				} else {
					fmt.Printf("  Removed %s:%s\n", srv.Host, srv.SyncPath)
				}
			}
		}

		// Remove log directory (lives outside the config dir on macOS).
		if logPath, err := config.LogPath(); err == nil {
			os.RemoveAll(filepath.Dir(logPath))
		}

		binaryPath, err := os.Executable()
		if err == nil {
			os.Remove(binaryPath)
		}

		fmt.Println("cpssh uninstalled.")
		return nil
	},
}

// shellSingleQuote wraps s in single quotes for safe inclusion in a remote
// shell command. Single quotes don't expand anything; an embedded ' is escaped
// by closing the quote, inserting a literal ', and reopening.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
