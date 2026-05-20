//go:build darwin

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

static long cpssh_change_count(void) {
    @autoreleasepool {
        return (long)[[NSPasteboard generalPasteboard] changeCount];
    }
}

static int cpssh_has_png(void) {
    @autoreleasepool {
        NSArray *types = [[NSPasteboard generalPasteboard] types];
        return [types containsObject:NSPasteboardTypePNG] ? 1 : 0;
    }
}

// Returns the post-write changeCount so the caller can advance its
// "seen" marker without an extra round trip.
// declareTypes: is used (not clearContents+writeObjects) because it produces a
// single change-count increment — two increments cause the watcher to race and
// re-detect our own write as a new user copy, triggering a sync loop.
// PNG is declared first (preferred type) but written last: Universal Clipboard
// appears to key off the most-recently-set data, so writing PNG last makes iOS
// apps receive the image rather than the text path.
static long cpssh_write_image_and_text(const void* imgData, int imgLen, const char* text) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb declareTypes:@[NSPasteboardTypePNG, NSPasteboardTypeString] owner:nil];
        NSString *str = [NSString stringWithUTF8String:text];
        [pb setString:str forType:NSPasteboardTypeString];
        NSData *data = [NSData dataWithBytes:imgData length:imgLen];
        [pb setData:data forType:NSPasteboardTypePNG];
        return (long)[pb changeCount];
    }
}
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

type systemClipboard struct {
	pngpaste string

	mu     sync.Mutex
	seenCC int64 // NSPasteboard changeCount we've already processed
}

func New() Clipboard {
	// LaunchAgents run with a stripped PATH that excludes /opt/homebrew/bin.
	// Resolve the full path at startup so polling works regardless of PATH.
	path, err := exec.LookPath("pngpaste")
	if err != nil {
		for _, p := range []string{"/opt/homebrew/bin/pngpaste", "/usr/local/bin/pngpaste"} {
			if _, serr := os.Stat(p); serr == nil {
				path = p
				break
			}
		}
	}
	if path == "" {
		log.Printf("clipboard: pngpaste not found — image sync disabled. Install: brew install pngpaste")
	}
	return &systemClipboard{pngpaste: path}
}

// WriteImageAndText writes both a PNG image and a text reference to the system
// pasteboard simultaneously. Terminal paste gives text; image apps get the PNG.
// Advances the watcher's seen changeCount so it doesn't re-detect this write.
func (s *systemClipboard) WriteImageAndText(imageData []byte, text string) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	var imgPtr unsafe.Pointer
	if len(imageData) > 0 {
		imgPtr = unsafe.Pointer(&imageData[0])
	}
	newCC := int64(C.cpssh_write_image_and_text(imgPtr, C.int(len(imageData)), cText))
	s.markSeen(newCC)
}

func (s *systemClipboard) markSeen(cc int64) {
	s.mu.Lock()
	if cc > s.seenCC {
		s.seenCC = cc
	}
	s.mu.Unlock()
}

func (s *systemClipboard) lastSeen() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seenCC
}

func (s *systemClipboard) WatchForImage(interval time.Duration) <-chan []byte {
	ch := make(chan []byte, 1)
	go func() {
		if s.pngpaste == "" {
			return
		}
		log.Printf("clipboard watching for images...")
		// Seed seenCC so we don't treat whatever's already on the pasteboard
		// at daemon start as a fresh user copy.
		s.markSeen(int64(C.cpssh_change_count()))
		tmp := filepath.Join(os.TempDir(), fmt.Sprintf("cpssh-clip-%d.png", os.Getpid()))
		for {
			time.Sleep(interval)
			cc := int64(C.cpssh_change_count())
			if cc == s.lastSeen() {
				continue
			}
			// Skip the pngpaste subprocess for text/link copies — clipboard
			// polls every 300ms and most changes aren't images.
			if C.cpssh_has_png() == 0 {
				s.markSeen(cc)
				continue
			}
			if err := exec.Command(s.pngpaste, tmp).Run(); err != nil {
				s.markSeen(cc)
				continue
			}
			data, err := os.ReadFile(tmp)
			os.Remove(tmp)
			if err != nil || len(data) == 0 {
				s.markSeen(cc)
				continue
			}
			s.markSeen(cc)
			log.Printf("clipboard: image detected (%d KB)", len(data)/1024)
			ch <- data
		}
	}()
	return ch
}
