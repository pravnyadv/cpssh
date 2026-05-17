package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Server struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user"`
	SSHKey   string `json:"ssh_key"`
	SyncPath string `json:"sync_path"`
}

type Settings struct {
	PollIntervalMs  int  `json:"poll_interval_ms"`
	MaxFileSizeKB   int  `json:"max_file_size_kb"`
	CompressAboveKB int  `json:"compress_above_kb"`
	KeepLastNFiles  int  `json:"keep_last_n_files"`
	Paused          bool `json:"paused"`
}

type Config struct {
	Servers  []Server `json:"servers"`
	Settings Settings `json:"settings"`
}

func DefaultConfig() *Config {
	return &Config{
		Servers: []Server{},
		Settings: Settings{
			PollIntervalMs:  300,
			MaxFileSizeKB:   2048,
			CompressAboveKB: 500,
			KeepLastNFiles:  10,
			Paused:          false,
		},
	}
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cpssh"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found — run: cpssh setup")
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	defaults := DefaultConfig().Settings
	if cfg.Settings.PollIntervalMs <= 0 {
		cfg.Settings.PollIntervalMs = defaults.PollIntervalMs
	}
	if cfg.Settings.MaxFileSizeKB <= 0 {
		cfg.Settings.MaxFileSizeKB = defaults.MaxFileSizeKB
	}
	if cfg.Settings.KeepLastNFiles <= 0 {
		cfg.Settings.KeepLastNFiles = defaults.KeepLastNFiles
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	path := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return nil
}

func (c *Config) AddServer(s Server) {
	c.Servers = append(c.Servers, s)
}

func PIDPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

// LogPath returns the log file path. It does NOT create the parent directory —
// callers that need to write to the file should call EnsureLogDir first.
// Keeping LogPath side-effect-free lets uninstall query it after removing the
// config dir without accidentally re-creating anything.
func LogPath() (string, error) {
	dir, err := logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cpssh.log"), nil
}

// EnsureLogDir creates the log file's parent directory if missing.
func EnsureLogDir() error {
	dir, err := logDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

func logDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// macOS: ~/Library/Logs/cpssh/ (standard, visible in Console.app)
	// Linux: ~/.config/cpssh/ (XDG)
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Logs", "cpssh"), nil
	}
	return filepath.Join(home, ".config", "cpssh"), nil
}

func counterPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "counter"), nil
}

const maxLocalImages = 10

// NextImageName returns the next image filename cycling img1.png–img10.png.
// Cycling caps local and remote storage at 10 files naturally.
func NextImageName() string {
	path, err := counterPath()
	if err != nil {
		return "img1.png"
	}

	var n int
	data, err := os.ReadFile(path)
	if err == nil {
		fmt.Sscanf(string(data), "%d", &n)
	}
	n = (n % maxLocalImages) + 1

	_ = os.WriteFile(path, []byte(fmt.Sprintf("%d", n)), 0600)
	return fmt.Sprintf("img%d.png", n)
}
