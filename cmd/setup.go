package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pravnyadv/cpssh/internal/config"
	"github.com/pravnyadv/cpssh/internal/daemon"
	"github.com/pravnyadv/cpssh/internal/server"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-time setup",
	RunE:  runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	checkDeps()

	fmt.Println("Welcome to cpssh setup!")
	fmt.Println("This will configure clipboard-to-SSH image sync.\n")

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	for {
		srv, err := promptServer()
		if err != nil {
			return err
		}

		fmt.Printf("Testing SSH connection to %s@%s...\n", srv.User, srv.Host)
		if err := server.TestConnection(srv); err != nil {
			fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
			return err
		}
		fmt.Println("Connection successful.")

		fmt.Println("Setting up remote server...")
		if err := server.Setup(srv); err != nil {
			return fmt.Errorf("remote setup: %w", err)
		}
		fmt.Println("Remote setup complete.")

		cfg.AddServer(srv)

		if !prompt("Add another server? [y/N]: ", false) {
			break
		}
	}

	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "/usr/local/bin/cpssh"
	}

	if !daemon.DaemonInstalled() {
		fmt.Println("Installing daemon...")
		if err := daemon.InstallDaemon(binaryPath); err != nil {
			return fmt.Errorf("install daemon: %w", err)
		}
		fmt.Println("Daemon installed and started.")
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("\ncpssh is running. Copy an image and it will sync to your server(s) automatically.")
	return nil
}

func promptServer() (config.Server, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("SSH server (user@host): ")
	raw, _ := reader.ReadString('\n')
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return config.Server{}, fmt.Errorf("server cannot be empty")
	}

	parts := strings.SplitN(raw, "@", 2)
	var user, host string
	if len(parts) == 2 {
		user, host = parts[0], parts[1]
	} else {
		host = parts[0]
		user = os.Getenv("USER")
	}

	sshKey := pickSSHKey(reader)

	fmt.Print("Remote sync path [$HOME/.cpssh]: ")
	syncPath, _ := reader.ReadString('\n')
	syncPath = strings.TrimSpace(syncPath)
	if syncPath == "" {
		syncPath = "$HOME/.cpssh"
	}
	if strings.HasPrefix(syncPath, "~/") {
		syncPath = "$HOME/" + syncPath[2:]
	} else if syncPath == "~" {
		syncPath = "$HOME"
	}

	return config.Server{
		Host:     host,
		User:     user,
		SSHKey:   sshKey,
		SyncPath: syncPath,
	}, nil
}

func pickSSHKey(reader *bufio.Reader) string {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	entries, _ := os.ReadDir(sshDir)

	var keys []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".pub") &&
			name != "known_hosts" && name != "config" &&
			name != "authorized_keys" {
			keys = append(keys, filepath.Join(sshDir, name))
		}
	}

	if len(keys) == 0 {
		fmt.Print("SSH key path: ")
		path, _ := reader.ReadString('\n')
		return strings.TrimSpace(path)
	}

	fmt.Println("Available SSH keys:")
	for i, k := range keys {
		fmt.Printf("  [%d] %s\n", i+1, k)
	}
	fmt.Printf("  [0] Enter path manually\n")
	fmt.Print("Pick a key [1]: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return keys[0]
	}
	if input == "0" {
		fmt.Print("SSH key path: ")
		path, _ := reader.ReadString('\n')
		return strings.TrimSpace(path)
	}

	var idx int
	fmt.Sscanf(input, "%d", &idx)
	if idx >= 1 && idx <= len(keys) {
		return keys[idx-1]
	}
	return keys[0]
}

func prompt(msg string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(msg)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

func checkDeps() {
	missing := []string{}
	if runtime.GOOS == "linux" {
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			if _, err := exec.LookPath("wl-paste"); err != nil {
				missing = append(missing, "wl-clipboard (install: sudo apt install wl-clipboard)")
			}
		} else {
			if _, err := exec.LookPath("xclip"); err != nil {
				missing = append(missing, "xclip (install: sudo apt install xclip)")
			}
		}
	}
	if _, err := exec.LookPath("ssh"); err != nil {
		missing = append(missing, "ssh")
	}
	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "Missing dependencies:")
		for _, m := range missing {
			fmt.Fprintf(os.Stderr, "  - %s\n", m)
		}
		os.Exit(1)
	}
}
