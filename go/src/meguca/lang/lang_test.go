package lang

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meguca/config"
)

func init() {
	if err := Load(); err != nil {
		panic(err)
	}
}

func TestGet(t *testing.T) {
	config.Set(config.Configs{
		Public: config.Public{
			DefaultLang: "en_GB",
		},
	})

	cases := [...]struct {
		name, cookie, header string
		out                  string
	}{
		{
			name: "default language",
			out:  "en_GB",
		},
		{
			name:   "default language set cookie",
			cookie: "en_GB",
			out:    "en_GB",
		},
		{
			name:   "other language cookie",
			cookie: "pt_BR",
			out:    "pt_BR",
		},
		{
			name:   "invalid language cookie",
			cookie: "bs_BS",
			out:    "en_GB",
		},
		{
			name:   "short locale header",
			header: "pt",
			out:    "pt_BR",
		},
		{
			name:   "long locale header",
			header: "pt-BR",
			out:    "pt_BR",
		},
		{
			name:   "invalid locale header",
			header: "bs",
			out:    "en_GB",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			if c.cookie != "" {
				cookie := http.Cookie{
					Name:  "lang",
					Value: c.cookie,
				}
				r.AddCookie(&cookie)
			}
			if c.header != "" {
				r.Header.Set("Accept-Language", c.header)
			}

			pack, err := Get(w, r)
			if err != nil {
				t.Fatal(err)
			}

			if pack.ID != c.out {
				t.Errorf("unexpected language pack id: %s : %s", c.out, pack.ID)
			}

			expected := strings.Replace(c.out, "_", "-", 1)
			if h := w.Header().Get("Content-Language"); h != expected {
				t.Errorf("unexpected response header: %s : %s", expected, h)
			}
		})
	}
}
