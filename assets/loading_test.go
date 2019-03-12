package assets

import (
	"os"
	"testing"
)

// Simply assert there is no panic
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
