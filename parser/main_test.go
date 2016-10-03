package parser

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
)

func init() {
	db.DBName = "meguca_test_parser"
	db.IsTest = true
	if err := db.LoadDB(); err != nil {
		panic(err)
	}
	config.Set(config.Configs{})
}

func logUnexpected(t *testing.T, expected, got interface{}) {
	t.Errorf("\nexpected: %#v\ngot:      %#v", expected, got)
}

func assertTableClear(t *testing.T, tables ...string) {
	if err := db.ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertInsert(t *testing.T, table string, doc interface{}) {
	if err := db.Insert(table, doc); err != nil {
		t.Fatal(err)
	}
}

func assertDeepEquals(t *testing.T, res, std interface{}) {
	if !reflect.DeepEqual(res, std) {
		logUnexpected(t, std, res)
	}
}

func genString(len int) string {
	var buf bytes.Buffer
	for i := 0; i < len; i++ {
		buf.WriteRune(rune(rand.Intn(128)))
	}
	return buf.String()
}

func TestStripPsuedoWhitespace(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in, out string
	}{
		{"without", "normal", "normal"},
		{"with", "h\u2000e\u200fl\u202al\u202fo\u205f\u206f", "hello"},
		{"unicode with", "日本\u2062語", "日本語"},
	}
	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if s := stripPsuedoWhitespace(c.in); s != c.out {
				logUnexpected(t, c.out, s)
			}
		})
	}
}
