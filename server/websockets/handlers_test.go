package websockets

import (
	"encoding/json"
	"testing"
)

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
		logUnexpected(t, std, msg)
	}
}
