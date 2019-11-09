package feeds

import (
	"os"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/test"
	"github.com/bakape/meguca/test/test_db"
)

func TestMain(m *testing.M) {
	err := config.Server.Load()
	if err != nil {
		panic(err)
	}
	close, err := db.LoadTestDB("feeds")
	if err != nil {
		panic(err)
	}

	code := m.Run()
	err = close()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func TestWriteMultipleToBuffer(t *testing.T) {
	t.Parallel()

	f := Feed{}
	f.write([]byte("a"))
	f.write([]byte("b"))

	const std = "33[\"a\",\"b\"]"
	if s := string(f.flush()); s != std {
		test.LogUnexpected(t, std, s)
	}
}

func TestHandleModeration(t *testing.T) {
	Clear()
	test_db.ClearTables(t, "boards")
	test_db.WriteSampleBoard(t)
	test_db.WriteSampleThread(t)
}
