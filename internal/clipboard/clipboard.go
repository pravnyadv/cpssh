package clipboard

import "time"

// Clipboard polls the system clipboard for image data.
type Clipboard interface {
	// WatchForImage emits PNG bytes each time a new image appears on the clipboard.
	WatchForImage(interval time.Duration) <-chan []byte
}
