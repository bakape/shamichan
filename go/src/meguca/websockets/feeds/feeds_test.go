package feeds

import (
	. "meguca/test"
	"testing"
)

func TestWriteMultipleToBuffer(t *testing.T) {
	t.Parallel()

	f := Feed{}
	f.write([]byte("a"))
	f.write([]byte("b"))

	const std = "33a\u0000b"
	if s := string(f.flush()); s != std {
		LogUnexpected(t, std, s)
	}
}
