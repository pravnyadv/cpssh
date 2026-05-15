package cmd

import (
	"os"
	"os/exec"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail the daemon log",
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath, err := config.LogPath()
		if err != nil {
			return err
		}
		tail := exec.Command("tail", "-f", logPath)
		tail.Stdout = os.Stdout
		tail.Stderr = os.Stderr
		return tail.Run()
	},
}
