package db

import (
	"fmt"
	"testing"
	"time"

	"reflect"

	"github.com/bakape/meguca/config"
	r "github.com/dancannon/gorethink"
)

func init() {
	DBName = "meguca_test"
	isTest = true
	if err := LoadDB(); err != nil {
		panic(err)
	}
}

func assertTableClear(t *testing.T, tables ...string) {
	if err := ClearTables(tables...); err != nil {
		t.Fatal(err)
	}
}

func assertInsert(t *testing.T, table string, doc interface{}) {
	if err := Insert(table, doc); err != nil {
		t.Fatal(err)
	}
}

func logUnexpected(t *testing.T, expected, got interface{}) {
	t.Errorf("\nexpected: %#v\ngot:      %#v", expected, got)
}

func TestVerifyVersion(t *testing.T) {
	assertTableClear(t, "main")
	assertInsert(t, "main", map[string]interface{}{
		"id":        "info",
		"dbVersion": dbVersion,
	})

	// Correct DB version
	if err := verifyDBVersion(); err != nil {
		t.Error(err)
	}

	q := GetMain("info").Update(map[string]int{
		"dbVersion": 0,
	})
	if err := Write(q); err != nil {
		t.Error(err)
	}

	// Incompatible DB version
	err := verifyDBVersion()
	if fmt.Sprint(err) != `incompatible RethinkDB database version: 0` {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestPopulateDB(t *testing.T) {
	assertTableClear(t, AllTables...)

	// Remove all indeces
	q := r.
		TableList().
		ForEach(func(table r.Term) r.Term {
			return r.
				Table(table).
				IndexList().
				ForEach(func(index r.Term) r.Term {
					return r.Table(table).IndexDrop(index)
				})
		})
	if err := Exec(q); err != nil {
		t.Fatal(err)
	}

	if err := populateDB(); err != nil {
		t.Fatal(err)
	}

	// Assert all tables exist
	t.Run("tables exist", func(t *testing.T) {
		t.Parallel()
		var missingTables []string
		q := r.Expr(AllTables).Difference(r.TableList()).Default([]string{})
		if err := All(q, &missingTables); err != nil {
			t.Error(err)
		}
		for _, table := range missingTables {
			t.Errorf("table '%s' not created", table)
		}
	})

	// Assert all secondary indeces exist
	for _, index := range secondaryIndeces {
		i := index // Capture variable
		name := fmt.Sprintf("index %s:%s", i.table, i.index)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var hasIndex bool
			q := r.Table(i.table).IndexList().Contains(i.index)
			if err := One(q, &hasIndex); err != nil {
				t.Fatal(err)
			}
			if !hasIndex {
				t.Fatalf(
					"no secondary index '%s' created for table '%s'",
					i.index,
					i.table,
				)
			}
		})
	}

	t.Run("info", func(t *testing.T) {
		t.Parallel()
		std := infoDocument{Document{"info"}, dbVersion, 0}
		var res infoDocument
		if err := One(GetMain("info"), &res); err != nil {
			t.Fatal(err)
		}
		if res != std {
			logUnexpected(t, std, res)
		}
	})

	t.Run("boardCtrs", func(t *testing.T) {
		t.Parallel()
		assertExists(t, GetAccount("admin"))
	})

	t.Run("admin account", func(t *testing.T) {
		t.Parallel()
		assertExists(t, GetMain("info"))
	})

	t.Run("config", func(t *testing.T) {
		t.Parallel()
		var conf config.Configs
		if err := One(GetMain("config"), &conf); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(conf, config.Defaults) {
			logUnexpected(t, config.Defaults, conf)
		}
	})
}

func assertExists(t *testing.T, q r.Term) {
	var exists bool
	if err := One(q.Eq(nil).Not(), &exists); err != nil {
		t.Error(err)
	}
	if !exists {
		t.Error("not found")
	}
}

func TestUpgrade14to15(t *testing.T) {
	assertTableClear(t, "main", "boards")
	assertInsert(t, "main", map[string]interface{}{
		"id":      "info",
		"version": 14,
	})
	assertInsert(t, "boards", config.BoardConfigs{
		ID: "a",
	})

	if err := upgrade14to15(); err != nil {
		t.Fatal(err)
	}

	t.Run("version", func(t *testing.T) {
		t.Parallel()
		var v int
		q := GetMain("info").Field("dbVersion")
		if err := One(q, &v); err != nil {
			t.Fatal(err)
		}
		if v != 15 {
			t.Fatalf("unexpected version: %d", v)
		}
	})

	t.Run("insert 'created' field", func(t *testing.T) {
		t.Parallel()
		var created time.Time
		q := r.Table("boards").Get("a").Field("created")
		if err := One(q, &created); err != nil {
			t.Fatal(err)
		}
		if !created.Before(time.Now()) {
			t.Fatalf("invalid timestamp: %v", created)
		}
	})
}
