package common

import (
	"encoding/json"
	"testing"

	. "github.com/bakape/meguca/test"
)

func TestCommandsMarshaling(t *testing.T) {
	cases := [...]struct {
		name string
		com  Command
	}{
		{"pyu", Command{
			Type: Pyu,
			Pyu:  1,
		}},
		{"pcount", Command{
			Type: Pcount,
			Pyu:  1,
		}},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			buf, err := json.Marshal(c.com)
			if err != nil {
				t.Fatal(err)
			}
			var res Command
			err = json.Unmarshal(buf, &res)
			if err != nil {
				t.Fatal(err)
			}
			AssertEquals(t, res, c.com)
		})
	}
}
