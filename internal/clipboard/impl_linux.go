//go:build linux

package clipboard

import (
	"crypto/sha256"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type systemClipboard struct {
	mu       sync.Mutex
	lastHash [32]byte
}

func New() Clipboard {
	return &systemClipboard{}
}

// WriteImageAndText writes the text reference to the clipboard. On Linux the
// image is not preserved alongside the text — xclip/wl-copy don't support
// setting multiple types in one call the way NSPasteboard does.
//
// Resets the dedup hash so a subsequent re-copy of the same image will be
// re-emitted instead of silently swallowed.
func (s *systemClipboard) WriteImageAndText(_ []byte, text string) {
	var cmd *exec.Cmd
	if isWayland() {
		cmd = exec.Command("wl-copy")
	} else {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		log.Printf("clipboard: write failed: %v", err)
	}
	s.mu.Lock()
	s.lastHash = [32]byte{}
	s.mu.Unlock()
}

func isWayland() bool { return os.Getenv("WAYLAND_DISPLAY") != "" }

func (s *systemClipboard) WatchForImage(interval time.Duration) <-chan []byte {
	ch := make(chan []byte, 1)
	go func() {
		log.Printf("clipboard watching for images...")
		for {
			time.Sleep(interval)
			data := readClipboardImage()
			if len(data) == 0 {
				continue
			}
			h := sha256.Sum256(data)
			s.mu.Lock()
			if h == s.lastHash {
				s.mu.Unlock()
				continue
			}
			s.lastHash = h
			s.mu.Unlock()
			log.Printf("clipboard: image detected (%d KB)", len(data)/1024)
			ch <- data
		}
	}()
	return ch
}

func readClipboardImage() []byte {
	if isWayland() {
		data, _ := exec.Command("wl-paste", "--type", "image/png").Output()
		return data
	}
	data, _ := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o").Output()
	return data
}
