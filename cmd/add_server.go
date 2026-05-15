package cmd

import (
	"fmt"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/pravnyadv/cpssh/internal/server"
	"github.com/spf13/cobra"
)

var addServerCmd = &cobra.Command{
	Use:   "add-server",
	Short: "Add another SSH server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		srv, err := promptServer()
		if err != nil {
			return err
		}

		fmt.Printf("Testing connection to %s@%s...\n", srv.User, srv.Host)
		if err := server.TestConnection(srv); err != nil {
			return err
		}

		if err := server.Setup(srv); err != nil {
			return err
		}

		cfg.AddServer(srv)
		return cfg.Save()
	},
}
