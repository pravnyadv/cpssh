//go:build linux

package clipboard

import goclipboard "golang.design/x/clipboard"

// WriteImageAndText on Linux just writes the text ref.
func WriteImageAndText(imageData []byte, textRef string) {
	goclipboard.Write(goclipboard.FmtText, []byte(textRef))
}
