package cmd

import (
	"fmt"
	"os"

	"github.com/pravnyadv/cpssh/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Run the clipboard watch loop (managed by launchd/systemd)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := daemon.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return nil
	},
}
