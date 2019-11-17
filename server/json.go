package server

import (
	"net/http"
	"strconv"
)

func setJSONHeaders(w http.ResponseWriter) {
	head := w.Header()
	for key, val := range vanillaHeaders {
		head.Set(key, val)
	}
	head.Set("Content-Type", "application/json")
}

// Validate the client's last N posts to display setting. To allow for better
// caching the only valid values are 5 and 50. 5 is for index-like thread
// previews and 50 is for short threads.
func detectLastN(r *http.Request) int {
	if q := r.URL.Query().Get("last"); q != "" {
		n, err := strconv.Atoi(q)
		if err == nil && (n == 100 || n == 5) {
			return n
		}
	}
	return 0
}

// // Serve a single post as JSON
// func servePost(w http.ResponseWriter, r *http.Request) {
// 	id, err := strconv.ParseUint(extractParam(r, "post"), 10, 64)
// 	if err != nil {
// 		httpError(w, r, common.StatusError{
// 			Err:  err,
// 			Code: 400,
// 		})
// 		return
// 	}

// 	post, err := db.GetPost(id)
// 	if err != nil {
// 		httpError(w, r, err)
// 		return
// 	}
// 	serveJSON(w, r, "", post)
// }

// // Serves thread page JSON
// func threadJSON(w http.ResponseWriter, r *http.Request) {
// 	var (
// 		page   int
// 		thread uint64
// 	)
// 	ok := func() (ok bool) {
// 		var err error
// 		thread, err = strconv.ParseUint(extractParam(r, "thread"), 10, 64)
// 		if err != nil {
// 			return
// 		}

// 		if s := r.URL.Query().Get("page"); s != "" {
// 			page, err = strconv.Atoi(s)
// 			if err != nil || page < -1 {
// 				return
// 			}
// 		}

// 		ok = true
// 		return
// 	}()
// 	if !ok {
// 		text404(w)
// 		return
// 	}

// 	httpError(w, r, func() (err error) {
// 		setJSONHeaders(w)
// 		return cache.Thread(w, r, thread, page)
// 	}())
// }

// func serveThreadUpdates(w http.ResponseWriter, r *http.Request) {
// 	err := func() (err error) {
// 		var data map[uint64]uint64
// 		err = decodeJSON(r, &data)
// 		if err != nil {
// 			return
// 		}

// 		diff, err := db.DiffThreadPostCounts(data)
// 		if err != nil {
// 			return
// 		}
// 		serveJSON(w, r, "", diff)
// 		return
// 	}()
// 	if err != nil {
// 		httpError(w, r, err)
// 	}
// }
