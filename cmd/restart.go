package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/pravnyadv/cpssh/internal/daemon"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidPath, _ := config.PIDPath()
		pid, running := readPID(pidPath)

		if running {
			if cfg, err := config.Load(); err == nil && cfg.Settings.Paused {
				cfg.Settings.Paused = false
				_ = cfg.Save()
				fmt.Println("Resumed.")
			}
			proc, _ := os.FindProcess(pid)
			// SIGTERM lets the daemon clean up its PID file before launchd
			// (KeepAlive) or systemd (Restart=always) brings it back.
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("could not stop daemon: %w", err)
			}
			fmt.Printf("Daemon stopped (pid %d). Waiting for restart", pid)
			for i := 0; i < 10; i++ {
				time.Sleep(500 * time.Millisecond)
				fmt.Print(".")
				if newPid, ok := readPID(pidPath); ok && newPid != pid {
					fmt.Printf(" done (pid %d)\n", newPid)
					return nil
				}
			}
			fmt.Println(" done")
			return nil
		}

		if !daemon.DaemonInstalled() {
			return fmt.Errorf("daemon not installed — run: cpssh setup")
		}
		fmt.Print("Starting daemon")
		if err := daemon.StartDaemon(); err != nil {
			return err
		}
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			fmt.Print(".")
			if _, ok := readPID(pidPath); ok {
				fmt.Println(" done")
				return nil
			}
		}
		fmt.Println(" done")
		return nil
	},
}
