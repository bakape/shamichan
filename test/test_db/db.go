// Database testing utility functions

package test_db

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/bakape/meguca/db"
	uuid "github.com/satori/go.uuid"
)

// Key pair used by clients to authenticate handshakes and file uploads
type KeyPair struct {
	PrivID uint64
	PubID  uuid.UUID
	Key    *rsa.PrivateKey
}

// Insert sample thread and return its ID
func InsertSampleThread(t *testing.T) (id uint64, keyPair KeyPair) {
	t.Helper()

	var err error
	keyPair.Key, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatal(err)
	}

	keyPair.PrivID, keyPair.PubID, _, err = db.RegisterPublicKey(
		x509.MarshalPKCS1PublicKey(&keyPair.Key.PublicKey),
	)
	id, err = db.InsertThread(db.ThreadInsertParams{
		Subject: "test",
		Tags:    []string{"animu", "mango"},
		PostInsertParamsCommon: db.PostInsertParamsCommon{
			PublicKey: &keyPair.PrivID,
			Body:      []byte("{}"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id not set")
	}
	return
}

func ClearTables(t testing.TB, tables ...string) {
	t.Helper()

	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}
