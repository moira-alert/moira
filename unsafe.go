package moira

import (
	"unsafe"
)

// UnsafeBytesToString converts source to string without copying.
func UnsafeBytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// UnsafeStringToBytes converts string to source without copying.
func UnsafeStringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
