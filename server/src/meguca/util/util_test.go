package util

import (
	"errors"
	"testing"

	. "meguca/test"
)

func TestWrapError(t *testing.T) {
	t.Parallel()

	err := errors.New("foo")
	wrapped := WrapError("bar", err)
	if s := wrapped.Error(); s != "bar: foo" {
		t.Fatalf("unexpected error: `%s`", s)
	}
}

func TestHashBuffer(t *testing.T) {
	t.Parallel()

	if h := HashBuffer([]byte{1, 2, 3}); h != "Uonfc331cyb83SJZevsfrA" {
		t.Fatalf("unexpected hash: %s", h)
	}
}

func TestWaterfall(t *testing.T) {
	// All pass
	var wasRun int
	fn := func() error {
		wasRun++
		return nil
	}
	if err := Waterfall(fn, fn); err != nil {
		t.Fatal(err)
	}
	if wasRun != 2 {
		t.Fatalf("wrong run number: %d", wasRun)
	}

	// 2nd function returns error
	wasRun = 0
	stdErr := errors.New("foo")
	fns := []func() error{
		fn,
		func() error {
			wasRun++
			return stdErr
		},
		fn,
	}
	if err := Waterfall(fns...); err != stdErr {
		UnexpectedError(t, err)
	}
	if wasRun != 2 {
		t.Fatalf("wrong run number: %d", wasRun)
	}
}
