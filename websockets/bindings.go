package websockets

// #cgo CFLAGS: -std=c11
// #cgo LDFLAGS: -Wl,--no-as-needed -ldl -lm
// #include "bindings.h"
// #include <stdlib.h>
// #include <string.h>
import "C"
import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"unsafe"

	"github.com/bakape/meguca/cache"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"github.com/go-playground/log"
	"github.com/jackc/pgx/v4"
	uuid "github.com/satori/go.uuid"
)

func init() {
	util.Hook("config.updated", func() (err error) {
		buf, err := json.Marshal(config.Get())
		if err != nil {
			return
		}
		return fromCError(C.ws_set_config(toWSBuffer(buf)))
	})
}

// Initialize module. Must be run after DB is online.
func Init() (err error) {
	buf, err := db.GetFeedData()
	if err != nil {
		return
	}
	return fromCError(C.ws_init(toWSBuffer(buf)))
}

// Construct byte slice from C pointer and size
func toSlice(ptr *C.uint8_t, size C.size_t) []byte {
	return *(*[]byte)(
		unsafe.Pointer(
			&reflect.SliceHeader{
				Data: uintptr(unsafe.Pointer(ptr)),
				Len:  int(size),
				Cap:  int(size),
			},
		),
	)
}

//export ws_thread_exists
func ws_thread_exists(id C.uint64_t, exists *bool) *C.char {
	_exists, err := db.ThreadExists(context.Background(), uint64(id))
	if err != nil {
		return C.CString(err.Error())
	}
	*exists = _exists
	return nil
}

//export ws_write_message
func ws_write_message(clientID C.uint64_t, msg C.WSRcBuffer) {
	// Spawning separate goroutine to not block the pulsar thread pool
	go func() {
		// Not using deferred unlock to prevent possible deadlocks between the
		// Go and Rust client collection mutexes. These must be freed as soon as
		// possible.
		clientsMu.RLock()
		c, ok := clients[uint64(clientID)]
		clientsMu.RUnlock()

		if ok {
			select {
			case c.send <- msg:
			case <-c.ctx.Done():
				// Client is dead - need to unreference in its stead
				C.ws_unref_message(msg.src)
			}
		} else {
			// No client, so unreference immediately
			C.ws_unref_message(msg.src)
		}
	}()
}

//export ws_close_client
func ws_close_client(clientID C.uint64_t, err *C.char) {
	// Not using deferred unlock to not block on channel send
	clientsMu.Lock()
	c, ok := clients[uint64(clientID)]
	clientsMu.Unlock()

	if ok {
		var e error
		if err != nil {
			e = errors.New(C.GoString(err))
		}
		select {
		case c.close <- e:
		case <-c.ctx.Done():
		}
	}
}

//export ws_insert_thread
func ws_insert_thread(
	subject *C.char,
	tags **C.char,
	tags_size C.size_t,
	public_key C.uint64_t,
	body C.WSBuffer,
	id *C.uint64_t,
) *C.char {
	tags_ := make([]string, int(tags_size))
	size := unsafe.Sizeof(subject)
	for i := range tags_ {
		tags_[i] = C.GoString(
			*(**C.char)(unsafe.Pointer(
				uintptr(unsafe.Pointer(tags)) + size*uintptr(i)),
			),
		)
	}

	pk := uint64(public_key)
	id_, err := db.InsertThread(
		context.Background(),
		db.ThreadInsertParams{
			Subject: C.GoString(subject),
			Tags:    tags_,
			PostInsertParamsCommon: db.PostInsertParamsCommon{
				PublicKey: &pk,
				Body:      toSlice(body.data, body.size),
			},
		},
	)
	if err != nil {
		return C.CString(err.Error())
	}
	*id = C.uint64_t(id_)

	cache.EvictThreadList()

	return nil
}

//export ws_register_public_key
func ws_register_public_key(
	pub_key C.WSBuffer,
	priv_id *C.uint64_t,
	pub_id *C.uint8_t, // UUID exposed to clients
	fresh *C.bool, // freshly registered (did not exist before this)
) *C.char {
	priv_id_, pub_id_, fresh_, err := db.RegisterPublicKey(
		toSlice(pub_key.data, pub_key.size),
	)
	if err != nil {
		return C.CString(err.Error())
	}
	*priv_id = C.uint64_t(priv_id_)
	C.memcpy(unsafe.Pointer(pub_id), unsafe.Pointer(&pub_id_[0]), 16)
	*fresh = C.bool(fresh_)
	return nil
}

//export ws_get_public_key
func ws_get_public_key(
	pub_id *C.uint8_t, // UUID exposed to clients; used for lookup
	priv_id *C.uint64_t,
	pub_key *C.WSBuffer, // Owned by caller and must be freed
) *C.char {
	var pub_id_ uuid.UUID
	C.memcpy(unsafe.Pointer(&pub_id_[0]), unsafe.Pointer(pub_id), 16)
	priv_id_, pub_key_, err := db.GetPubKey(pub_id_)
	if err != nil {
		if err == pgx.ErrNoRows {
			return C.CString("unknown public key ID")
		}
		return C.CString(err.Error())
	}
	*priv_id = C.uint64_t(priv_id_)
	pub_key.data = (*C.uint8_t)(C.CBytes(pub_key_))
	pub_key.size = C.size_t(len(pub_key_))
	return nil
}

//export ws_get_post_parenthood
func ws_get_post_parenthood(
	id C.uint64_t,
	thread *C.uint64_t,
	page *C.uint32_t,
) *C.char {
	thread_, page_, err := db.GetPostParenthood(uint64(id))
	if err != nil {
		return C.CString(err.Error())
	}
	*thread = C.uint64_t(thread_)
	*page = C.uint32_t(page_)
	return nil
}

//export ws_increment_spam_score
func ws_increment_spam_score(pub_key C.uint64_t, score C.uint64_t) {
	db.IncrementSpamScore(uint64(pub_key), uint64(score))
}

//export ws_write_open_post_bodies
func ws_write_open_post_bodies(buf C.WSBuffer) *C.char {
	err := db.WriteOpenPostBodies(toSlice(buf.data, buf.size))
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export ws_validate_captcha
func ws_validate_captcha(
	id *C.uint8_t, // Always 64 bytes
	solution *C.uint8_t,
	size C.size_t,
) *C.char {
	// TODO: user-specific captchas after captchouli port to Postgres
	return nil
}

//export ws_log_error
func ws_log_error(err *C.char) {
	log.Errorf("websockets: %s", C.GoString(err))
}

// Cast []bytes to WSBuffer without copy
func toWSBuffer(buf []byte) C.WSBuffer {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	return C.WSBuffer{
		(*C.uint8_t)(unsafe.Pointer(h.Data)),
		C.size_t(h.Len),
	}
}

// Register image insertion into an open post.
//
// image: JSON-encoded inserted image data
func InsertImage(thread, post uint64, img common.Image) (err error) {
	buf, err := json.Marshal(img)
	if err != nil {
		return
	}
	return fromCError(C.ws_insert_image(
		C.uint64_t(thread),
		C.uint64_t(post),
		toWSBuffer(buf),
	))
}

// Cast any owned C error to Go and free it
func fromCError(errC *C.char) (err error) {
	if errC != nil {
		err = errors.New(C.GoString(errC))
	}
	C.free(unsafe.Pointer(errC))
	return
}
