package db

import (
	. "meguca/test"
	"testing"
)

func TestThreadBools(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)
	writeSampleThread(t)

	locked, err := CheckThreadLocked(1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, false, locked)

	nonLive, err := CheckThreadNonLive(1)
	if err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, false, nonLive)
}
