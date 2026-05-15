//go:build darwin

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#import <stdlib.h>

// Writes PNG data AND a text ref to the clipboard simultaneously.
// Terminal paste gets the text; image apps (Discord, Preview) get the PNG.
void writeImageAndText(const unsigned char* imgBytes, int imgLen, const char* text) {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    [pb clearContents];
    NSData *png = [NSData dataWithBytes:imgBytes length:imgLen];
    [pb setData:png forType:NSPasteboardTypePNG];
    [pb setString:[NSString stringWithUTF8String:text] forType:NSPasteboardTypeString];
}
*/
import "C"
import "unsafe"

// WriteImageAndText sets both PNG and a text reference on the clipboard.
func WriteImageAndText(imageData []byte, textRef string) {
	if len(imageData) == 0 {
		return
	}
	cText := C.CString(textRef)
	defer C.free(unsafe.Pointer(cText))
	C.writeImageAndText(
		(*C.uchar)(unsafe.Pointer(&imageData[0])),
		C.int(len(imageData)),
		cText,
	)
}
