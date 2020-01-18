package test_assets

import (
	"testing"

	"github.com/bakape/meguca/imager/assets"
)

// Create image diretories and return a function that deletes them
func SetupImageDirs(t *testing.T) func() {
	t.Helper()

	if err := assets.CreateDirs(); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := assets.DeleteDirs(); err != nil {
			t.Fatal(err)
		}
	}
}
