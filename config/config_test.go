package config

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/bakape/meguca/test"
)

func TestSetGet(t *testing.T) {
	conf := Configs{}
	conf.Hats = true

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

func TestMarshalPublicBoardJSON(t *testing.T) {
	b := BoardConfigs{
		CodeTags: true,
		PostParseConfigs: PostParseConfigs{
			ReadOnly: true,
		},
		Spoiler: "foo.png",
		Title:   "Animu",
		Banners: []string{},
	}
	std := `
{
	"banners":[],
	"codeTags":true,
	"forcedAnon":false,
	"hashCommands":false,
	"notice":"",
	"readOnly":true,
	"rules":"",
	"spoiler":"foo.png",
	"textOnly":false,
	"title":"Animu"
}`
	std = strings.Replace(std, "\t", "", -1)
	std = strings.Replace(std, "\n", "", -1)

	data, err := b.MarshalPublicJSON()
	if err != nil {
		t.Fatal(err)
	}
	if s := string(data); s != std {
		LogUnexpected(t, std, s)
	}
}
