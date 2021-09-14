package converters

import (
	"bytes"
	"reflect"
	"unsafe"
)

func UnsafeBytesToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func UnsafeStringToBytes(str string) []byte {
	return *(*[]byte)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&str))))
}

func UnsafeUTF16BytesToString(bs []byte) string {
	return UnsafeBytesToString(bytes.Trim(bs, "\x00"))
}
