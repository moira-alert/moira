package moira

import (
	"reflect"
	"unsafe"
)

// UnsafeBytesToString converts source to string without copying
func UnsafeBytesToString(b []byte) string {
	hdr := *(*reflect.SliceHeader)(unsafe.Pointer(&b))
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{
		Data: hdr.Data,
		Len:  hdr.Len,
	}))
}

// UnsafeStringToBytes converts string to source without copying
func UnsafeStringToBytes(s string) []byte {
	header := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: header.Data,
		Len:  header.Len,
		Cap:  header.Len,
	}))
}
