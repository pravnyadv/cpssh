package server

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pravnyadv/cpssh/internal/config"
)

func baseArgs(s config.Server) []string {
	args := []string{
		"-i", s.SSHKey,
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if s.Port != 0 {
		args = append(args, "-p", fmt.Sprintf("%d", s.Port))
	}
	return args
}

// TestConnection verifies SSH access to the server.
func TestConnection(s config.Server) error {
	args := append(baseArgs(s), "-o", "BatchMode=yes", fmt.Sprintf("%s@%s", s.User, s.Host), "echo ok")
	out, err := exec.Command("ssh", args...).Output()
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
	args := append(baseArgs(s), fmt.Sprintf("%s@%s", s.User, s.Host), remoteCmd)
	return exec.Command("ssh", args...).Run()
}
