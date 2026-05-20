package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withHomeDir redirects HOME so functions that depend on UserHomeDir read/write
// inside a per-test scratch directory.
func withHomeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func TestLoad_MissingConfig(t *testing.T) {
	withHomeDir(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected error when config is missing")
	} else if !strings.Contains(err.Error(), "cpssh setup") {
		t.Errorf("error should mention setup, got: %v", err)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	withHomeDir(t)

	cfg := DefaultConfig()
	cfg.AddServer(Server{
		Host:     "example.com",
		Port:     2222,
		User:     "alice",
		SSHKey:   "/home/alice/.ssh/id_ed25519",
		SyncPath: "$HOME/.cpssh",
	})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Servers) != 1 {
		t.Fatalf("want 1 server, got %d", len(got.Servers))
	}
	if got.Servers[0] != cfg.Servers[0] {
		t.Errorf("server roundtrip mismatch:\n want %+v\n  got %+v", cfg.Servers[0], got.Servers[0])
	}
	if got.Settings != cfg.Settings {
		t.Errorf("settings roundtrip mismatch:\n want %+v\n  got %+v", cfg.Settings, got.Settings)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	withHomeDir(t)

	if err := DefaultConfig().Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	path, _ := ConfigPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("config file perms: want 0600, got %o", mode)
	}
}

func TestLoad_FillsZeroDefaults(t *testing.T) {
	home := withHomeDir(t)

	// Write a config that omits most settings — Load should backfill defaults.
	dir := filepath.Join(home, ".config", "cpssh")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw := `{"servers":[],"settings":{}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Settings.PollIntervalMs != 300 {
		t.Errorf("PollIntervalMs default: want 300, got %d", cfg.Settings.PollIntervalMs)
	}
	if cfg.Settings.MaxFileSizeKB != 10240 {
		t.Errorf("MaxFileSizeKB default: want 10240, got %d", cfg.Settings.MaxFileSizeKB)
	}
	if cfg.Settings.KeepLastNFiles != 10 {
		t.Errorf("KeepLastNFiles default: want 10, got %d", cfg.Settings.KeepLastNFiles)
	}
}

func TestLoad_NegativeSettingsTreatedAsZero(t *testing.T) {
	home := withHomeDir(t)
	dir := filepath.Join(home, ".config", "cpssh")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw := `{"servers":[],"settings":{"poll_interval_ms":-5,"max_file_size_kb":-1,"keep_last_n_files":0}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Settings.PollIntervalMs <= 0 {
		t.Errorf("negative PollIntervalMs should be replaced with default, got %d", cfg.Settings.PollIntervalMs)
	}
	if cfg.Settings.MaxFileSizeKB <= 0 {
		t.Errorf("negative MaxFileSizeKB should be replaced with default, got %d", cfg.Settings.MaxFileSizeKB)
	}
	if cfg.Settings.KeepLastNFiles <= 0 {
		t.Errorf("zero KeepLastNFiles should be replaced with default, got %d", cfg.Settings.KeepLastNFiles)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	home := withHomeDir(t)
	dir := filepath.Join(home, ".config", "cpssh")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestNextImageName_CyclesOneToTen(t *testing.T) {
	withHomeDir(t)
	// Ensure config dir exists so counterPath can be written.
	if err := DefaultConfig().Save(); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"img1.png", "img2.png", "img3.png", "img4.png", "img5.png",
		"img6.png", "img7.png", "img8.png", "img9.png", "img10.png",
		"img1.png", "img2.png",
	}
	for i, w := range want {
		got, _ := NextImageName()
		if got != w {
			t.Errorf("call %d: want %s, got %s", i, w, got)
		}
	}
}

func TestNextImageName_RecoversFromCorruptCounter(t *testing.T) {
	withHomeDir(t)
	if err := DefaultConfig().Save(); err != nil {
		t.Fatal(err)
	}

	path, err := counterPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("garbage"), 0600); err != nil {
		t.Fatal(err)
	}

	if got, _ := NextImageName(); got != "img1.png" {
		t.Errorf("after corrupt counter, want img1.png, got %s", got)
	}
}

func TestAddServer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AddServer(Server{Host: "a"})
	cfg.AddServer(Server{Host: "b"})
	if len(cfg.Servers) != 2 || cfg.Servers[1].Host != "b" {
		t.Errorf("AddServer didn't append correctly: %+v", cfg.Servers)
	}
}

func TestLogPath_DoesNotCreateDir(t *testing.T) {
	home := withHomeDir(t)
	if _, err := LogPath(); err != nil {
		t.Fatalf("LogPath: %v", err)
	}
	// LogPath should be side-effect-free; the parent dir must not exist yet.
	dir, _ := logDir()
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("LogPath unexpectedly created %s", dir)
	}
	// Sanity: EnsureLogDir actually creates it.
	if err := EnsureLogDir(); err != nil {
		t.Fatalf("EnsureLogDir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("EnsureLogDir failed to create %s: %v", dir, err)
	}
	_ = home
}
