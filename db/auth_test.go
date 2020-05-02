package db

import (
	"math/rand"
	"testing"

	"github.com/bakape/meguca/test"
	uuid "github.com/satori/go.uuid"
)

func TestRegisterPublicKey(t *testing.T) {
	var pubKey [1 << 10]byte
	_, err := rand.Read(pubKey[:])
	if err != nil {
		t.Fatal(err)
	}

	// Initial insert
	privID1, pubID1, fresh, err := RegisterPublicKey(pubKey[:])
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, fresh, true)

	// Existing key
	privID2, pubID2, fresh, err := RegisterPublicKey(pubKey[:])
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, privID1, privID2)
	test.AssertEquals(t, pubID1, pubID2)
	test.AssertEquals(t, fresh, false)

	privID3, pubKey3, err := GetPubKey(pubID1)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, privID3, privID1)
	test.AssertEquals(t, pubKey[:], pubKey3)

	// Different key
	_, err = rand.Read(pubKey[:])
	if err != nil {
		t.Fatal(err)
	}
	_, _, fresh, err = RegisterPublicKey(pubKey[:])
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, fresh, true)
}

// Does not actually insert a valid public key, but that is fine as they are
// only read in the websockets module
func insertSamplePubKey(t *testing.T) (privID uint64, pubID uuid.UUID) {
	t.Helper()

	var key [1 << 10]byte
	_, err := rand.Read(key[:])
	if err != nil {
		t.Fatal(err)
	}
	privID, pubID, _, err = RegisterPublicKey(key[:])
	if err != nil {
		t.Fatal(err)
	}
	return
}
