package db

import (
	"testing"

	"github.com/bakape/meguca/assets"
	. "github.com/bakape/meguca/test"
)

func TestBanners(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)

	std := []assets.File{
		{
			Data: []byte{1, 2, 3},
			Mime: "data/meme",
			Hash: "Uonfc331cyb83SJZevsfrA",
		},
	}

	err := SetBanners("a", std)
	if err != nil {
		t.Fatal(err)
	}
	err = loadBanners()
	if err != nil {
		t.Fatal(err)
	}
	err = updateAssets("banners", assets.Banners.Set)("a")
	if err != nil {
		t.Fatal(err)
	}

	banner, ok := assets.Banners.Get("a", 0)
	if !ok {
		t.Fatal("banner not saved")
	}
	AssertEquals(t, banner, std[0])

	err = SetBanners("a", []assets.File{})
	if err != nil {
		t.Fatal(err)
	}
	err = updateAssets("banners", assets.Banners.Set)("a")
	if err != nil {
		t.Fatal(err)
	}
	_, ok = assets.Banners.Get("a", 0)
	if ok {
		t.Fatal("banner not deleted")
	}
}

func TestLoaadingAnimations(t *testing.T) {
	assertTableClear(t, "boards")
	writeSampleBoard(t)

	std := assets.File{
		Data: []byte{1, 2, 3},
		Mime: "data/meme",
		Hash: "Uonfc331cyb83SJZevsfrA",
	}

	err := SetLoadingAnimation("a", std)
	if err != nil {
		t.Fatal(err)
	}
	err = loadLoadingAnimations()
	if err != nil {
		t.Fatal(err)
	}

	f := assets.Loading.Get("a")
	AssertEquals(t, f, std)

	err = SetLoadingAnimation("a", assets.File{})
	if err != nil {
		t.Fatal(err)
	}
	err = updateAssets("loading_animations", setLoadingAnimation)("a")
	if err != nil {
		t.Fatal(err)
	}
	assets.Loading.Get("a")
}
