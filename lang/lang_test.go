package lang

import (
	"testing"
)

func init() {
	Dir = "."
}

func TestLoad(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
}
