package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFrontpageRedirect(t *testing.T) {
	config = serverConfigs{}
	config.Frontpage = "./test/frontpage.html"
	server := httptest.NewServer(http.HandlerFunc(redirectToDefault))
	defer server.Close()
	res, err := http.Get(server.URL)
	fatal(t, err)
	frontpage, err := ioutil.ReadAll(res.Body)
	fatal(t, err)
	fatal(t, res.Body.Close())
	html := string(frontpage)
	if html != "<!doctype html><html></html>\n" {
		t.Fatal(html)
	}
}

func fatal(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefaultBoardRedirect(t *testing.T) {
	config = serverConfigs{}
	config.Boards.Default = "a"
	req, err := http.NewRequest("GET", "/", nil)
	fatal(t, err)
	w := httptest.NewRecorder()
	redirectToDefault(w, req)
	if w.Code != 302 && string(w.Header().Get("Location")[0]) != "/a/" {
		t.Fatalf("%#v\n", w)
	}
}
