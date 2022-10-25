package moira

import (
	"reflect"
	"unsafe"
)

// UnsafeBytesToString converts source to string without copying
func UnsafeBytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// UnsafeStringToBytes converts string to source without copying
func UnsafeStringToBytes(s string) []byte {
	var b []byte
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data = (*reflect.StringHeader)(unsafe.Pointer(&s)).Data
	bh.Len = len(s)
	bh.Cap = len(s)

	return b
}
