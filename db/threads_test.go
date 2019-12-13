package db

import "testing"

func TestInsertThread(t *testing.T) {
	id, err := InsertThread("test", []string{"anime", "mango"}, genToken(t))
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id not set")
	}
}
