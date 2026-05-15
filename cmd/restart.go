package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		pidPath, _ := config.PIDPath()
		pid, running := readPID(pidPath)
		if !running {
			return fmt.Errorf("daemon not running")
		}
		proc, _ := os.FindProcess(pid)
		if err := proc.Kill(); err != nil {
			return fmt.Errorf("could not kill daemon: %w", err)
		}
		fmt.Printf("Daemon stopped (pid %d). Launchd will restart it automatically.\n", pid)

		fmt.Print("Waiting for restart")
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
	},
}
