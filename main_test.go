package main

import (
	"os"
	"testing"
)

// Simple test, to see if the server starts
func TestServerStart(t *testing.T) {
	os.Args = []string{os.Args[0], "init"}
	defer func() {
		if err := recover(); err != nil {
			t.Fatal(err)
		}
	}()
	main()
}
