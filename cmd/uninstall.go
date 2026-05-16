package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/pravnyadv/cpssh/internal/daemon"
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
				sshArgs := []string{
					"-i", srv.SSHKey,
					"-o", "ConnectTimeout=10",
					"-o", "StrictHostKeyChecking=accept-new",
				}
				if srv.Port != 0 {
					sshArgs = append(sshArgs, "-p", fmt.Sprintf("%d", srv.Port))
				}
				sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", srv.User, srv.Host), fmt.Sprintf(`rm -rf "%s"`, srv.SyncPath))
				c := exec.Command("ssh", sshArgs...)
				if err := c.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not remove %s:%s — %v\n", srv.Host, srv.SyncPath, err)
				} else {
					fmt.Printf("  Removed %s:%s\n", srv.Host, srv.SyncPath)
				}
			}
		}

		fmt.Println("cpssh uninstalled.")
		return nil
	},
}
