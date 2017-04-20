package feeds

import (
	. "meguca/test"
	"testing"
)

func TestWriteMultipleToBuffer(t *testing.T) {
	t.Parallel()

	f := Feed{}
	f.Write([]byte("a"))
	f.Write([]byte("b"))

	const std = "33a\u0000b"
	if s := string(f.Flush()); s != std {
		LogUnexpected(t, std, s)
	}
}
