package config

import (
	"bytes"
	"testing"

	. "github.com/bakape/meguca/test"
)

func TestSetGet(t *testing.T) {
	conf := Configs{
		Public: Public{
			Hats: true,
		},
	}

	if err := Set(conf); err != nil {
		t.Fatal(err)
	}
	AssertDeepEquals(t, Get(), &conf)

	json, hash := GetClient()
	if json == nil {
		t.Fatal("client json not set")
	}
	if hash == "" {
		t.Fatal("hash not set")
	}
}

func TestSetGetClient(t *testing.T) {
	std := []byte{1, 2, 3}
	hash := "foo"
	SetClient(std, hash)

	json, jsonHash := GetClient()
	if !bytes.Equal(json, std) {
		LogUnexpected(t, std, json)
	}
	if jsonHash != hash {
		LogUnexpected(t, hash, jsonHash)
	}
}

func TestSetGetBoards(t *testing.T) {
	std := []string{"a", "b", "c"}
	SetBoards(std)
	AssertDeepEquals(t, GetBoards(), std)
}

func TestSetGetAddRemoveBoardConfigs(t *testing.T) {
	std := BoardConfigs{
		ID: "a",
		BoardPublic: BoardPublic{
			Spoilers: true,
		},
	}
	SetBoards([]string{"a", "x"})

	if err := SetBoardConfigs(std); err != nil {
		t.Fatal(err)
	}
	conf := GetBoardConfigs("a")
	if conf.Hash == "" {
		t.Fatal("no hash generated")
	}
	if conf.JSON == nil {
		t.Fatal("no JSON generated")
	}
	AssertDeepEquals(t, conf.BoardConfigs, std)

	RemoveBoard("a")
	AddBoard("c")
	AssertDeepEquals(t, GetBoardConfigs("a"), BoardConfContainer{})
	AssertDeepEquals(t, GetBoards(), []string{"x", "c"})
}
