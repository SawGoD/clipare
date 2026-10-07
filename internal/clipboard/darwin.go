//go:build darwin && cgo

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>
static long change_count(void) { @autoreleasepool { return [[NSPasteboard generalPasteboard] changeCount]; } }
static char* read_text(int *status) { @autoreleasepool {
 NSString *s = [[NSPasteboard generalPasteboard] stringForType:NSPasteboardTypeString];
 if (!s) { *status=0; return NULL; }
 NSData *d = [s dataUsingEncoding:NSUTF8StringEncoding];
 if ([d length]>1048576) { *status=2; return NULL; }
 char *p=malloc([d length]+1); if(!p){*status=3;return NULL;}
 memcpy(p,[d bytes],[d length]); p[[d length]]=0; *status=1; return p;
} }
static int write_text(const char *p, long n) { @autoreleasepool {
 NSString *s = [[NSString alloc] initWithBytes:p length:n encoding:NSUTF8StringEncoding];
 if (!s) return 0;
 NSPasteboard *pb = [NSPasteboard generalPasteboard];
 [pb clearContents]; BOOL ok=[pb setString:s forType:NSPasteboardTypeString]; [s release]; return ok;
} }
*/
import "C"
import (
	"context"
	"errors"
	"time"
	"unsafe"
)

type Native struct{}

func New() (Backend, error) { return &Native{}, nil }
func (*Native) Read() (string, bool, error) {
	var status C.int
	p := C.read_text(&status)
	if p != nil {
		defer C.free(unsafe.Pointer(p))
	}
	switch status {
	case 0:
		return "", false, nil
	case 2:
		return "", false, ErrTooLarge
	case 3:
		return "", false, errors.New("clipboard allocation failed")
	}
	return C.GoString(p), true, nil
}
func (*Native) Write(s string) error {
	p := C.CString(s)
	defer C.free(unsafe.Pointer(p))
	if C.write_text(p, C.long(len(s))) == 0 {
		return errors.New("clipboard write failed")
	}
	return nil
}
func (*Native) Watch(ctx context.Context, ch chan<- struct{}) error {
	last := C.change_count()
	t := time.NewTicker(400 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			current := C.change_count()
			if current != last {
				last = current
				notify(ctx, ch)
			}
		}
	}
}
