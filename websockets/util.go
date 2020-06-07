package websockets

// #include "bindings.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"errors"
	"reflect"
	"unsafe"

	"github.com/bakape/meguca/db"
)

// Construct byte slice from WSBuffer without copy.
//
// unsafe: returned instance has the same lifetime as ptr
func toSlice(buf C.WSBuffer) []byte {
	if buf.data == nil || buf.size == 0 {
		return nil
	}
	return *(*[]byte)(
		unsafe.Pointer(
			&reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(buf.data)),
				Len:  int(buf.size),
				Cap:  int(buf.size),
			},
		),
	)
}

// Construct string slice from WSBuffer without copy.
//
// unsafe: returned instance has the same lifetime as ptr
func toString(buf C.WSBuffer) string {
	if buf.data == nil || buf.size == 0 {
		return ""
	}
	return *(*string)(
		unsafe.Pointer(
			&reflect.StringHeader{
				Data: uintptr(unsafe.Pointer(buf.data)),
				Len:  int(buf.size),
			},
		),
	)
}

// Construct string slice from WSBuffer with copying
func toStringCopy(buf C.WSBuffer) string {
	return string(toSlice(buf))
}

// Cast []bytes to WSBuffer without copy
func toWSBuffer(buf []byte) C.WSBuffer {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	return C.WSBuffer{
		(*C.uint8_t)(unsafe.Pointer(h.Data)),
		C.size_t(h.Len),
	}
}

// Cast any owned C error to Go and free it
func fromCError(errC *C.char) (err error) {
	if errC != nil {
		err = errors.New(C.GoString(errC))
	}
	C.free(unsafe.Pointer(errC))
	return
}

// Map optional WSBuffer to string pointer
func mapOptional(buf C.WSBuffer) *string {
	s := toString(buf)
	if s == "" {
		return nil
	}
	return &s
}

func makePostInsertParamsCommon(
	public_key C.uint64_t,
	name, trip, body C.WSBuffer,
) db.PostInsertParamsCommon {
	public_key_ := uint64(public_key)
	return db.PostInsertParamsCommon{
		PublicKey: &public_key_,
		Name:      mapOptional(name),
		Trip:      mapOptional(trip),
		Body:      toSlice(body),
	}
}
