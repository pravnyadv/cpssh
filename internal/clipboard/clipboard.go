package clipboard

import "time"

// Clipboard polls the system clipboard for image data and writes back
// image-plus-text replacements after a successful sync.
type Clipboard interface {
	// WatchForImage emits PNG bytes each time a new image appears on the clipboard.
	WatchForImage(interval time.Duration) <-chan []byte

	// WriteImageAndText replaces the clipboard with the image plus a text
	// reference. The implementation also advances its internal "seen" marker
	// so the next poll does not re-detect the write as user activity.
	WriteImageAndText(imageData []byte, text string)
}
