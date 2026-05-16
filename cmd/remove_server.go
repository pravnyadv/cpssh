package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/spf13/cobra"
)

var removeServerCmd = &cobra.Command{
	Use:   "remove-server",
	Short: "Remove a configured server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Servers) == 0 {
			fmt.Println("No servers configured.")
			return nil
		}

		fmt.Println("Configured servers:")
		for i, s := range cfg.Servers {
			fmt.Printf("  [%d] %s (%s)\n", i+1, serverAddr(s), s.SyncPath)
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Remove server number: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var idx int
		fmt.Sscanf(input, "%d", &idx)
		if idx < 1 || idx > len(cfg.Servers) {
			return fmt.Errorf("invalid selection")
		}

		removed := cfg.Servers[idx-1]
		cfg.Servers = append(cfg.Servers[:idx-1], cfg.Servers[idx:]...)

		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Removed %s\n", serverAddr(removed))
		return nil
	},
}
