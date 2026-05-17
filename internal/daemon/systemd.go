//go:build linux

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const systemdUnit = `[Unit]
Description=cpssh daemon

[Service]
ExecStart=%s daemon
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
`

func unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", "cpssh.service"), nil
}

func InstallDaemon(binaryPath string) error {
	path, err := unitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	content := fmt.Sprintf(systemdUnit, binaryPath)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "cpssh").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func StartDaemon() error {
	if out, err := exec.Command("systemctl", "--user", "start", "cpssh").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl start: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func UninstallDaemon() error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", "cpssh").Run()
	path, err := unitPath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func DaemonInstalled() bool {
	path, err := unitPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
