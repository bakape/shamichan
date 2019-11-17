package config

import (
	"bytes"
	"testing"

	"github.com/bakape/meguca/test"
)

func TestSetGet(t *testing.T) {
	Clear()
	conf := Configs{
		Public: Public{
			Mature: true,
		},
	}

	if err := Set(conf); err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, Get(), &conf)

	json, hash := GetClient()
	if json == nil {
		t.Fatal("client json not set")
	}
	if hash == "" {
		t.Fatal("hash not set")
	}
}

func TestSetGetClient(t *testing.T) {
	Clear()
	std := []byte{1, 2, 3}
	hash := "foo"
	SetClient(std, hash)

	json, jsonHash := GetClient()
	if !bytes.Equal(json, std) {
		test.LogUnexpected(t, std, json)
	}
	if jsonHash != hash {
		test.LogUnexpected(t, hash, jsonHash)
	}
}
