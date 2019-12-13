package db

import (
	"testing"

	"github.com/bakape/meguca/test"
)

func TestInsertThread(t *testing.T) {
	id, err := InsertThread("test", []string{"anime", "mango"}, genToken(t))
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id not set")
	}

	exists, err := ThreadExists(id)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, exists, true)

	exists, err = ThreadExists(456636351)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, exists, false)
}
