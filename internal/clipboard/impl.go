package clipboard

import (
	"crypto/sha256"
	"log"
	"time"

	"golang.design/x/clipboard"
)

type systemClipboard struct{}

func New() Clipboard {
	return &systemClipboard{}
}

func (s *systemClipboard) WatchForImage(interval time.Duration) <-chan []byte {
	ch := make(chan []byte, 1)
	go func() {
		if err := clipboard.Init(); err != nil {
			log.Printf("clipboard init error: %v", err)
			return
		}
		log.Printf("clipboard watching for images...")
		var lastHash [32]byte
		for {
			time.Sleep(interval)
			data := clipboard.Read(clipboard.FmtImage)
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
