//go:build darwin

package clipboard

import (
	"crypto/sha256"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
		tmp := filepath.Join(os.TempDir(), "cpssh-clip.png")
		var lastHash [32]byte
		for {
			time.Sleep(interval)
			if err := exec.Command("pngpaste", tmp).Run(); err != nil {
				continue // no image on clipboard
			}
			data, err := os.ReadFile(tmp)
			os.Remove(tmp)
			if err != nil || len(data) == 0 {
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
