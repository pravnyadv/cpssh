package daemon

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pravnyadv/cpssh/internal/clipboard"
	"github.com/pravnyadv/cpssh/internal/config"
	cpssync "github.com/pravnyadv/cpssh/internal/sync"
)

// Run starts the clipboard watch loop. Blocks until SIGTERM/SIGINT.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := writePID(); err != nil {
		return err
	}
	defer removePID()

	if err := config.EnsureLogDir(); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logPath, _ := config.LogPath()
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Println("daemon started")
	go cpssync.WarmUp(cfg)

	cb := clipboard.New()
	interval := time.Duration(cfg.Settings.PollIntervalMs) * time.Millisecond
	images := cb.WatchForImage(interval)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-stop:
			log.Println("daemon stopped")
			return nil
		case data := <-images:
			cfg, err = config.Load()
			if err != nil {
				log.Printf("config reload error: %v", err)
				continue
			}
			if cfg.Settings.Paused {
				continue
			}
			if len(cfg.Servers) == 0 {
				continue
			}
			maxBytes := int64(cfg.Settings.MaxFileSizeKB) * 1024
			if int64(len(data)) > maxBytes {
				log.Printf("image too large (%d KB), skipping", len(data)/1024)
				continue
			}
			if !hasActiveSSHSession() {
				log.Printf("no active SSH session, skipping sync")
				continue
			}
			log.Printf("image captured (%d KB), syncing to %d server(s)", len(data)/1024, len(cfg.Servers))
			if remotePath := cpssync.SyncToAll(cfg, data); remotePath != "" {
				display := "[" + strings.ReplaceAll(remotePath, "$HOME/", "~/") + "]"
				cb.WriteImageAndText(data, display)
				log.Printf("clipboard: wrote image + %s", display)
			}
		}
	}
}

func writePID() error {
	path, err := config.PIDPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0600)
}

func removePID() {
	path, _ := config.PIDPath()
	os.Remove(path)
}
