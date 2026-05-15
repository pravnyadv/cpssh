package server

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pravnyadv/cpssh/internal/config"
)

// TestConnection verifies SSH access to the server.
func TestConnection(s config.Server) error {
	cmd := exec.Command("ssh", "-i", s.SSHKey,
		"-o", "ConnectTimeout=10",
		"-o", "BatchMode=yes",
		fmt.Sprintf("%s@%s", s.User, s.Host),
		"echo ok")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	if !strings.Contains(string(out), "ok") {
		return fmt.Errorf("unexpected SSH response")
	}
	return nil
}

// Setup creates the remote sync directory with correct permissions.
func Setup(s config.Server) error {
	cmd := fmt.Sprintf(`mkdir -p "%s" && chmod 700 "%s"`, s.SyncPath, s.SyncPath)
	if err := runSSH(s, cmd); err != nil {
		return fmt.Errorf("create remote dir: %w", err)
	}
	return nil
}

func runSSH(s config.Server, remoteCmd string) error {
	cmd := exec.Command("ssh", "-i", s.SSHKey,
		fmt.Sprintf("%s@%s", s.User, s.Host),
		remoteCmd)
	return cmd.Run()
}
