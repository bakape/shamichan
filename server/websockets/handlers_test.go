package websockets

import (
	"encoding/json"
	"testing"

	. "github.com/bakape/meguca/test"
)

func marshalJSON(t testing.TB, msg interface{}) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecodeMessage(t *testing.T) {
	t.Parallel()

	// Unparsable message
	var msg syncRequest
	assertErrorPrefix(t, decodeMessage([]byte{0}, &msg), invalidCharacter)

	// Valid message
	std := syncRequest{
		Thread: 20,
		Board:  "a",
	}
	data := marshalJSON(t, std)
	if err := decodeMessage(data, &msg); err != nil {
		t.Fatal(err)
	}
	if msg != std {
		LogUnexpected(t, std, msg)
	}
}
