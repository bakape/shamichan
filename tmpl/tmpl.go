// Package tmpl compiles HTML templates to be used for rendering pages during
// dynamic content insertion
package tmpl

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"hash"
	"io"
	"io/ioutil"
	"meguca/config"
	"meguca/lang"
	"os"
	"strings"
)

type templateStore struct {
	parts []string
	hash  string
}

type templateMap map[string]map[string]templateStore

// Resources exports temolates and their hashes by language
var Resources templateMap

// Compile reads template HTML from disk, injext dynamic variables, hashes and
// exports them
func Compile() error {
	// Only one for now, but there will be more later
	raw := map[string]string{}
	for _, name := range []string{"index"} {
		file, err := ioutil.ReadFile("./tmpl/html/" + name + ".html")
		if err != nil {
			return errors.Wrap(err, 1)
		}
		raw[name] = string(file)
	}
	if err := indexTemplate(raw["index"]); err != nil {
		return err
	}
	return nil
}

// ClientHash is the combined, shortened MD5 hash of all client files
var ClientHash string

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate(tmpl string) error {
	vars := map[string]string{}
	js, err := json.Marshal(config.ClientConfig)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	vars["config"] = string(js)
	vars["navigation"] = boardNavigation()
	hash, err1 := hashClientFiles()
	if err1 != nil {
		return err1
	}
	vars["hash"] = hash
	ClientHash = hash

	fields := []string{"name", "email", "options", "indentity", "faq",
		"schedule", "feedback", "onlineCounter", "Not synced"}
	for _, ln := range config.Config.Lang.Enabled {
		vars["lang"] = ln

		// Copy all translated template strings
		for _, field := range fields {
			vars[field] = lang.Langs[ln].Tmpl[field]
		}
	}
	return nil
}

func hashClientFiles() (string, error) {
	files := []string{}
	args := [][2]string{
		{"./www/css", ".css"},
		{"./www/js", ".js"},
		{"./www/js/vendor", ".js"},
		{"./www/js/es5", ".js"},
		{"./www/js/es6", ".js"},
		{"./www/js/lang", ".js"},
	}
	for _, pair := range args {
		contents, err := scanDir(pair[0], pair[1])
		if err != nil {
			return "", errors.Wrap(err, 0)
		}
		files = append(files, contents...)
	}

	hasher := md5.New()
	for _, file := range files {
		if err := hashFile(file, hasher); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))[:16], nil
}

// scanDir returns files from a folder, that end with the provided extension
func scanDir(path string, suffix string) ([]string, error) {
	files, err := ls(path)
	if err != nil {
		return nil, err
	}
	filtered := []string{}
	for _, file := range files {
		if strings.HasSuffix(file, suffix) {
			filtered = append(filtered, path+"/"+file)
		}
	}
	return filtered, nil
}

// ls returns the contents of a directory
func ls(path string) ([]string, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer dir.Close()
	files, err1 := dir.Readdirnames(0)
	if err1 != nil {
		return nil, errors.Wrap(err1, 0)
	}
	return files, nil
}

// hashFile reads a file from disk and pipes it into the hashing reader
func hashFile(path string, hasher hash.Hash) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()
	if _, err := io.Copy(hasher, file); err != nil {
		return errors.Wrap(err, 0)
	}
	return nil
}

// boardNavigation renders interboard navigation we put in the top banner
func boardNavigation() (html string) {
	html = `<b id="navTop">[`

	// Actual boards
	boards := config.Config.Boards.Enabled
	for i, board := range boards {
		if board == config.Config.Boards.Staff {
			continue
		}
		if i > 0 {
			html += ` / `
		}
		html += fmt.Sprintf(`<a href="../%v/" class="history">%v</a>`, board,
			board)
	}

	// Add custom URLs to board navigation
	for _, link := range config.Config.Boards.Psuedo {
		html += fmt.Sprintf(` / <a href="%v">%v</a>`, link[1], link[0])
	}
	html += `]</b>`
	return
}
