//go:build darwin

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// StandardOutPath/StandardErrorPath are intentionally omitted — the daemon
// opens the log file directly in Go (see daemon.Run). Letting launchd also
// write to it would interleave Go log lines with raw panic output.
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cpssh</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>daemon</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
`

type plistVars struct {
	BinaryPath string
}

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.cpssh.plist"), nil
}

func InstallDaemon(binaryPath string) error {
	path, err := plistPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, plistVars{BinaryPath: binaryPath}); err != nil {
		return err
	}

	target := launchTarget()
	_ = exec.Command("launchctl", "bootout", target, path).Run()
	if out, err := exec.Command("launchctl", "bootstrap", target, path).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func StartDaemon() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	target := launchTarget()
	_ = exec.Command("launchctl", "bootout", target, path).Run()
	if out, err := exec.Command("launchctl", "bootstrap", target, path).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func UninstallDaemon() error {
	path, err := plistPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "bootout", launchTarget(), path).Run()
	return os.Remove(path)
}

func launchTarget() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func DaemonInstalled() bool {
	path, err := plistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
