package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"strconv"

	"github.com/bakape/meguca/db"
)

const (
	notFound = "404 Not found\n"
)

func newRequest(url string) *http.Request {
	return httptest.NewRequest("GET", url, nil)
}

func newPair(url string) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), newRequest(url)
}

func assertCode(t *testing.T, rec *httptest.ResponseRecorder, std int) {
	if rec.Code != std {
		t.Errorf("unexpected status code: %d : %d", std, rec.Code)
	}
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

func logUnexpected(t *testing.T, expected, got interface{}) {
	t.Errorf("\nexpected: %#v\ngot:      %#v", expected, got)
}

func assertDeepEquals(t *testing.T, res, std interface{}) {
	if !reflect.DeepEqual(res, std) {
		logUnexpected(t, std, res)
	}
}

func assertEtag(t *testing.T, rec *httptest.ResponseRecorder, etag string) {
	if s := rec.Header().Get("ETag"); s != etag {
		t.Errorf("unexpected etag: %s : %s", etag, s)
	}
}

func assertBody(t *testing.T, rec *httptest.ResponseRecorder, body string) {
	if s := rec.Body.String(); s != body {
		t.Errorf("unexpected response body: %s : %s", body, s)
	}
}

func assertHeaders(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	h map[string]string,
) {
	for key, val := range h {
		if s := rec.Header().Get(key); s != val {
			t.Errorf("unexpected header %s value: %s : %s", key, val, s)
		}
	}
}

func marshalJSON(t *testing.T, msg interface{}) []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func dummyLog(n int) [][]byte {
	log := make([][]byte, n)
	for i := 0; i < n; i++ {
		log[i] = []byte{1}
	}
	return log
}

func TestSetHeaders(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	const etag = "foo"
	headers := map[string]string{
		"X-Frame-Options": "sameorigin",
		"Cache-Control":   "max-age=0, must-revalidate",
		"Expires":         "Fri, 01 Jan 1990 00:00:00 GMT",
		"ETag":            etag,
	}
	setHeaders(rec, etag)
	assertHeaders(t, rec, headers)
}

func TestCompareEtag(t *testing.T) {
	t.Parallel()

	rec, req := newPair("/")
	const etag = "foo"
	req.Header.Set("If-None-Match", etag)
	if pageEtag(rec, req, etag) {
		t.Error("unexpected match")
	}

	rec, req = newPair("/")
	headers := map[string]string{
		"ETag":          etag,
		"Cache-Control": "max-age=0, must-revalidate",
	}
	if !pageEtag(rec, req, etag) {
		t.Error("match expected")
	}
	assertHeaders(t, rec, headers)
}

func TestText404(t *testing.T) {
	t.Parallel()

	rec, req := newPair("/lalala/")
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 404)
	assertBody(t, rec, notFound)
}

func TestText500(t *testing.T) {
	t.Parallel()

	rec, req := newPair("/")
	req.RemoteAddr = "::1"
	text500(rec, req, errors.New("foo"))
	assertCode(t, rec, 500)
	assertBody(t, rec, "500 Internal server error: foo\n")
}

func TestText40X(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		code int
		fn   func(http.ResponseWriter, error)
		msg  string
	}{
		{400, text400, "Bad request"},
		{403, text403, "Forbidden"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(strconv.Itoa(c.code), func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			c.fn(rec, errors.New("foo"))
			assertCode(t, rec, c.code)
			assertBody(t, rec, fmt.Sprintf("%d %s: foo\n", c.code, c.msg))
		})
	}
}
