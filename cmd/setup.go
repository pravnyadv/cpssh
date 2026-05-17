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
	fmt.Println("This will configure clipboard-to-SSH image sync.")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	for {
		srv, err := promptServer()
		if err != nil {
			return err
		}

		if containsServer(cfg.Servers, srv) {
			fmt.Printf("%s is already configured, skipping.\n", serverAddr(srv))
		} else {
			fmt.Printf("Testing SSH connection to %s...\n", serverAddr(srv))
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
		}

		if !prompt("Add another server? [y/N]: ", false) {
			break
		}
	}

	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "/usr/local/bin/cpssh"
	}

	if !isStandardBinPath(binaryPath) {
		fmt.Printf("Note: daemon will run from %s — if you move this binary, re-run cpssh setup.\n", binaryPath)
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

	var port int
	if hostPort := strings.SplitN(host, ":", 2); len(hostPort) == 2 {
		host = hostPort[0]
		fmt.Sscanf(hostPort[1], "%d", &port)
	}

	if host == "" {
		return config.Server{}, fmt.Errorf("host cannot be empty")
	}
	if user == "" {
		return config.Server{}, fmt.Errorf("user cannot be empty — use user@host format")
	}

	portDefault := 22
	if port != 0 {
		portDefault = port
	}
	fmt.Printf("Port [%d]: ", portDefault)
	portInput, _ := reader.ReadString('\n')
	portInput = strings.TrimSpace(portInput)
	if portInput != "" {
		fmt.Sscanf(portInput, "%d", &port)
	} else {
		port = portDefault
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
		Port:     port,
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
		return readKeyPath(reader)
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
		return readKeyPath(reader)
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

func containsServer(servers []config.Server, s config.Server) bool {
	for _, existing := range servers {
		if existing.Host == s.Host && existing.Port == s.Port {
			return true
		}
	}
	return false
}

func isStandardBinPath(path string) bool {
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		"/usr/local/bin", "/usr/bin", "/opt/homebrew/bin",
		filepath.Join(home, "bin"), filepath.Join(home, ".local", "bin"),
	} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func serverAddr(s config.Server) string {
	if s.Port != 0 && s.Port != 22 {
		return fmt.Sprintf("%s@%s:%d", s.User, s.Host, s.Port)
	}
	return fmt.Sprintf("%s@%s", s.User, s.Host)
}

func readKeyPath(reader *bufio.Reader) string {
	fmt.Print("SSH key path: ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	if info, err := os.Stat(path); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: key not found: %s\n", path)
	} else if info.Mode().Perm()&0077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: key permissions are too open — run: chmod 400 %s\n", path)
	}
	return path
}

func checkDeps() {
	missing := []string{}
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("pngpaste"); err != nil {
			missing = append(missing, "pngpaste (install: brew install pngpaste)")
		}
	}
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
