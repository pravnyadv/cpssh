//go:build linux

package clipboard

import (
	"crypto/sha256"
	"log"
	"os"
	"os/exec"
	"time"
)

type systemClipboard struct{}

func New() Clipboard {
	return &systemClipboard{}
}

func (s *systemClipboard) WatchForImage(interval time.Duration) <-chan []byte {
	ch := make(chan []byte, 1)
	go func() {
		log.Printf("clipboard watching for images...")
		var lastHash [32]byte
		for {
			time.Sleep(interval)
			data := readClipboardImage()
			if len(data) == 0 {
				continue
			}
			h := sha256.Sum256(data)
			if h == lastHash {
				continue
			}
			lastHash = h
			log.Printf("clipboard: image detected (%d KB)", len(data)/1024)
			ch <- data
		}
	}()
	return ch
}

func readClipboardImage() []byte {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		data, _ := exec.Command("wl-paste", "--type", "image/png").Output()
		return data
	}
	data, _ := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o").Output()
	return data
}
