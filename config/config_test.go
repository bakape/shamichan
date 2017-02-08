package config

import (
	"bytes"
	"testing"

	. "github.com/bakape/meguca/test"
)

func TestGetAllBoard(t *testing.T) {
	t.Parallel()
	AssertDeepEquals(t, GetBoardConfigs("all"), AllBoardConfigs)
}

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
	Clear()
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

func TestGetBoards(t *testing.T) {
	ClearBoards()

	_, err := SetBoardConfigs(BoardConfigs{
		ID: "a",
	})
	if err != nil {
		t.Fatal(err)
	}

	AssertDeepEquals(t, GetBoards(), []string{"a"})
}

func TestSetGetAddRemoveBoardConfigs(t *testing.T) {
	ClearBoards()
	std := BoardConfigs{
		ID: "a",
		BoardPublic: BoardPublic{
			ForcedAnon: true,
		},
	}

	changed, err := SetBoardConfigs(std)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("configs not changed")
	}

	conf := GetBoardConfigs("a")
	if conf.Hash == "" {
		t.Fatal("no hash generated")
	}
	if conf.JSON == nil {
		t.Fatal("no JSON generated")
	}
	AssertDeepEquals(t, conf.BoardConfigs, std)
	if !IsBoard("a") {
		t.Fatal("board does not exist")
	}
	AssertDeepEquals(t, len(GetAllBoardConfigs()), 1)

	RemoveBoard("a")
	AssertDeepEquals(t, GetBoardConfigs("a"), BoardConfContainer{})
	if IsBoard("a") {
		t.Fatal("board not deleted")
	}
}

func TestSetMatchingBoardConfigs(t *testing.T) {
	ClearBoards()

	conf := BoardConfigs{
		ID: "a",
		BoardPublic: BoardPublic{
			ForcedAnon: true,
		},
	}

	for i := 0; i < 2; i++ {
		changed, err := SetBoardConfigs(conf)
		if err != nil {
			t.Fatal(err)
		}
		expected := i == 0
		if changed != expected {
			LogUnexpected(t, expected, changed)
		}
	}
}

func TestSetDifferentBoardConfigs(t *testing.T) {
	ClearBoards()

	conf := BoardConfigs{
		ID: "a",
		BoardPublic: BoardPublic{
			ForcedAnon: true,
		},
	}

	testBoardConfChange(t, conf)
	conf.Notice = "foo"
	testBoardConfChange(t, conf)
}

func testBoardConfChange(t *testing.T, conf BoardConfigs) {
	changed, err := SetBoardConfigs(conf)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
}

func TestGetBoardTitles(t *testing.T) {
	ClearBoards()

	conf := [...]BoardConfigs{
		{
			ID: "g",
			BoardPublic: BoardPublic{
				Title: "Techloligy",
			},
		},
		{
			ID: "a",
			BoardPublic: BoardPublic{
				Title: "Animu & Mango",
			},
		},
	}
	for _, c := range conf {
		if _, err := SetBoardConfigs(c); err != nil {
			t.Fatal(err)
		}
	}

	AssertDeepEquals(t, GetBoardTitles(), BoardTitles{
		{
			ID:    "a",
			Title: "Animu & Mango",
		},
		{
			ID:    "g",
			Title: "Techloligy",
		},
	})
}
