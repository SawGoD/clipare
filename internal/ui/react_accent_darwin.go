//go:build darwin && cgo

package ui

/*
#include <stdlib.h>
char *clipare_react_accent(void);
*/
import "C"
import "unsafe"

func reactSystemAccent() string {
	p := C.clipare_react_accent()
	defer C.free(unsafe.Pointer(p))
	return C.GoString(p)
}
