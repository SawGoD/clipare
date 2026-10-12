//go:build darwin && cgo

package history

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static CFStringRef cfstr(const char *s) { return CFStringCreateWithCString(NULL, s, kCFStringEncodingUTF8); }

static OSStatus keychain_load_buf(const char *service, const char *account, unsigned char *out, int outcap, int *outlen) {
	CFStringRef s = cfstr(service), a = cfstr(account);
	if (!s || !a) { if (s) CFRelease(s); if (a) CFRelease(a); return errSecAllocate; }
	const void *keys[] = {kSecClass, kSecAttrService, kSecAttrAccount, kSecReturnData};
	const void *values[] = {kSecClassGenericPassword, s, a, kCFBooleanTrue};
	CFDictionaryRef query = CFDictionaryCreate(NULL, keys, values, 4, NULL, NULL);
	CFDataRef data = NULL;
	OSStatus status = SecItemCopyMatching(query, (CFTypeRef *)&data);
	CFRelease(query);CFRelease(s);CFRelease(a);
	if (status == errSecSuccess) {
		CFIndex length = CFDataGetLength(data);
		if (length > outcap) { CFRelease(data); return errSecBufferTooSmall; }
		memcpy(out, CFDataGetBytePtr(data), length);
		*outlen = (int)length;
		CFRelease(data);
	}
	return status;
}

static OSStatus keychain_save_buf(const char *service, const char *account, const unsigned char *key, int keylen) {
	CFStringRef s = cfstr(service), a = cfstr(account);
	CFDataRef data = CFDataCreate(NULL, key, keylen);
	if (!s || !a || !data) { if (s) CFRelease(s); if (a) CFRelease(a); if (data) CFRelease(data); return errSecAllocate; }
	const void *qkeys[] = {kSecClass, kSecAttrService, kSecAttrAccount};
	const void *qvalues[] = {kSecClassGenericPassword, s, a};
	CFDictionaryRef query = CFDictionaryCreate(NULL, qkeys, qvalues, 3, NULL, NULL);
	const void *ukeys[] = {kSecValueData};
	const void *uvalues[] = {data};
	CFDictionaryRef attrs = CFDictionaryCreate(NULL, ukeys, uvalues, 1, NULL, NULL);
	OSStatus status = SecItemUpdate(query, attrs);
	CFRelease(attrs);CFRelease(query);
	if (status == errSecItemNotFound) {
		const void *akeys[] = {kSecClass, kSecAttrService, kSecAttrAccount, kSecValueData};
		const void *avalues[] = {kSecClassGenericPassword, s, a, data};
		CFDictionaryRef add = CFDictionaryCreate(NULL, akeys, avalues, 4, NULL, NULL);
		status = SecItemAdd(add, NULL);
		CFRelease(add);
	}
	CFRelease(s);CFRelease(a);CFRelease(data);
	return status;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"unsafe"
)

type keychainKeystore struct{}

func (keychainKeystore) Load() ([]byte, error) {
	service := C.CString(keyService)
	defer C.free(unsafe.Pointer(service))
	account := C.CString(keyAccount)
	defer C.free(unsafe.Pointer(account))
	out := make([]byte, keySize)
	var length C.int
	status := C.keychain_load_buf(service, account, (*C.uchar)(unsafe.Pointer(&out[0])), C.int(len(out)), &length)
	if status == C.errSecItemNotFound {
		return nil, os.ErrNotExist
	}
	if status != C.errSecSuccess {
		return nil, fmt.Errorf("keychain read failed: %d", int(status))
	}
	if int(length) != keySize {
		return nil, errors.New("history key has unexpected size")
	}
	return out, nil
}

func (keychainKeystore) Save(key []byte) error {
	if len(key) != keySize {
		return errors.New("history key must contain 32 bytes")
	}
	service := C.CString(keyService)
	defer C.free(unsafe.Pointer(service))
	account := C.CString(keyAccount)
	defer C.free(unsafe.Pointer(account))
	if status := C.keychain_save_buf(service, account, (*C.uchar)(unsafe.Pointer(&key[0])), C.int(len(key))); status != C.errSecSuccess {
		return fmt.Errorf("keychain write failed: %d", int(status))
	}
	return nil
}

func init() { defaultKeystore = keychainKeystore{} }
