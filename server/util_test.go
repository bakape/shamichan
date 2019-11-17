package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRequest(url string) *http.Request {
	return httptest.NewRequest("GET", url, nil)
}

func newPair(url string) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), newRequest(url)
}

func assertCode(t *testing.T, rec *httptest.ResponseRecorder, std int) {
	t.Helper()

	if rec.Code != std {
		t.Errorf("unexpected status code: %d : %d", std, rec.Code)

		var body string
		if rec.HeaderMap.Get("Content-Encoding") == "gzip" {
			var decoded bytes.Buffer
			gzr, err := gzip.NewReader(rec.Body)
			if err != nil {
				return
			}
			_, err = io.Copy(&decoded, gzr)
			if err != nil {
				return
			}
			err = gzr.Close()
			if err != nil {
				return
			}
			body = decoded.String()
		} else {
			body = rec.Body.String()
		}

		t.Logf("body: %s", body)
	}
}

func assertEtag(t *testing.T, rec *httptest.ResponseRecorder, etag string) {
	t.Helper()
	if s := rec.Header().Get("ETag"); s != etag {
		t.Errorf("unexpected etag: %s : %s", etag, s)
	}
}

func assertBody(t *testing.T, rec *httptest.ResponseRecorder, body string) {
	t.Helper()
	if s := rec.Body.String(); s != body {
		const f = "unexpected response body:\nexpected: `%s`\ngot:      `%s`"
		t.Errorf(f, body, s)
	}
}

func assertHeaders(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	h map[string]string,
) {
	t.Helper()
	for key, val := range h {
		if s := rec.Header().Get(key); s != val {
			t.Errorf("unexpected header %s value: %s : %s", key, val, s)
		}
	}
}

func marshalJSON(t *testing.T, msg interface{}) []byte {
	t.Helper()

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestText404(t *testing.T) {
	rec, req := newPair("/happi/")
	router.ServeHTTP(rec, req)
	assertCode(t, rec, 404)
	assertBody(t, rec, "404 not found\n")
}
