package imager

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ErikDubbelboer/gspt"
	"github.com/bakape/meguca/imager/assets"
	"github.com/bakape/meguca/imager/config"
	"github.com/bakape/meguca/imager/db"
	"github.com/jessevdk/go-flags"
)

func main() {
	err := func() (err error) {
		_, err = flags.Parse(&config.Server)
		if err != nil {
			return
		}

		// Censor DB connection string, if any
		args := make([]string, 0, len(os.Args))
		for i := 0; i < len(os.Args); i++ {
			arg := os.Args[i]
			// To match all of -d --d -database --database
			if strings.HasSuffix(arg, "-d") ||
				strings.HasSuffix(arg, "-database") {
				args = append(args, arg, "****")
				i++ // Jump to args after password
			} else {
				args = append(args, arg)
			}
		}
		gspt.SetProcTitle(strings.Join(args, " "))

		err = parallel(db.LoadDB, assets.CreateDirs)
		if err != nil {
			return
		}

		return startWebServer()
	}()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Parallel executes functions in parallel. The first error is returned, if any.
func parallel(fns ...func() error) error {
	ch := make(chan error, len(fns)) // Don't leak goroutines on error
	for i := range fns {
		fn := fns[i]
		go func() {
			ch <- fn()
		}()
	}

	for range fns {
		if err := <-ch; err != nil {
			return err
		}
	}

	return nil
}

func startWebServer() (err error) {
	go func() {
		// Bind pprof to random localhost-only address
		http.ListenAndServe("localhost:0", nil)
	}()

	addr := config.Server.Address

	postOnly := func(inner http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			inner(w, r)
		}
	}

	http.Handle("/upload", postOnly(NewImageUpload))
	http.Handle("/upload-hash", postOnly(UploadImageHash))
	http.Handle(
		"/health-check",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("God's in His heaven, all's right with the world"))
		}),
	)

	s := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}

	// Stop server on SIGTERM and propagate errors
	errCh := make(chan error)
	go func() {
		term := make(chan os.Signal, 1)
		signal.Notify(term, syscall.SIGTERM)
		<-term
		errCh <- s.Shutdown(context.Background())
	}()
	go func() {
		err := http.ListenAndServe(addr, http.DefaultServeMux)
		if err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	var w strings.Builder
	w.WriteString("listening on http")
	prettyAddr := addr
	if len(addr) != 0 && addr[0] == ':' {
		prettyAddr = "127.0.0.1" + prettyAddr
	}
	fmt.Fprintf(&w, "://%s", prettyAddr)
	log.Println(w.String())

	return <-errCh
}
