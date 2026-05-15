package cmd

import (
	"fmt"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Stop watching clipboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.Settings.Paused {
			fmt.Println("Already paused.")
			return nil
		}
		cfg.Settings.Paused = true
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println("cpssh paused.")
		return nil
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume watching clipboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if !cfg.Settings.Paused {
			fmt.Println("Already running.")
			return nil
		}
		cfg.Settings.Paused = false
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Println("cpssh resumed.")
		return nil
	},
}
