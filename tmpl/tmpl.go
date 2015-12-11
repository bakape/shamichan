// Package tmpl compiles HTML templates to be used for rendering pages during
// dynamic content insertion
package tmpl

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dchest/htmlmin"
	"hash"
	"html/template"
	"io"
	"io/ioutil"
	"meguca/config"
	"meguca/util"
	"os"
	"strings"
)

var throw = util.Throw

// Template stores the static part of HTML templates and the corresponding
// truncated MD5 hash of said template
type Template struct {
	Parts []string
	Hash  string
}

// TemplateMap stores all available templates
type TemplateMap map[string]Template

// Resources exports temolates and their hashes by language
var Resources TemplateMap

// Compile reads template HTML from disk, injext dynamic variables, hashes and
// exports them
func Compile() {
	// Only one for now, but there will be more later
	raw := map[string]string{}
	for _, name := range []string{"index"} {
		file, err := ioutil.ReadFile("./tmpl/html/" + name + ".html")
		throw(err)
		raw[name] = string(file)
	}
	indexTemplate(raw["index"])
}

// ClientHash is the combined, shortened MD5 hash of all client files
var ClientHash string

type templateVars struct {
	Config                                       template.JS
	Navigation, ClientHash, ConfigHash, MediaURL string
	IsMobile                                     bool
}

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate(raw string) {
	vars := templateVars{
		ConfigHash: config.ConfigHash,
		MediaURL:   config.Config.Hard.HTTP.Media,
	}
	js, err := json.Marshal(config.ClientConfig)
	throw(err)
	vars.Config = template.JS(js)
	vars.Navigation = boardNavigation()
	hash := hashClientFiles()
	vars.ClientHash = hash
	ClientHash = hash

	tmpl, err1 := template.New("index").Parse(raw)
	throw(err1)
	Resources = TemplateMap{}
	buildTemplate(tmpl, vars)
}

// hashClientFiles reads all client files and produces a truncated MD5 hash.
// Used for versioning in query strings for transparent client version
// transition.
func hashClientFiles() string {
	// Gather all files
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
		files = append(files, scanDir(pair[0], pair[1])...)
	}

	// Read all files into the hashing function
	hasher := md5.New()
	for _, file := range files {
		hashFile(file, hasher)
	}
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// scanDir returns files from a folder, that end with the provided extension
func scanDir(path string, suffix string) (filtered []string) {
	for _, file := range ls(path) {
		if strings.HasSuffix(file, suffix) {
			filtered = append(filtered, path+"/"+file)
		}
	}
	return
}

// ls returns the contents of a directory
func ls(path string) []string {
	dir, err := os.Open(path)
	throw(err)
	defer dir.Close()
	files, err1 := dir.Readdirnames(0)
	throw(err1)
	return files
}

// hashFile reads a file from disk and pipes it into the hashing reader
func hashFile(path string, hasher hash.Hash) {
	file, err := os.Open(path)
	throw(err)
	defer file.Close()
	_, err1 := io.Copy(hasher, file)
	throw(err1)
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

// buildTemplate constructs the HTML template array, minifies and hashes it
func buildTemplate(tmpl *template.Template, vars templateVars) {
	// Rigt now the desktop and mobile templates are almost identical. This will
	// change, when we get a dedicated mobile GUI.
	for _, kind := range []string{"desktop", "mobile"} {
		vars.IsMobile = kind == "mobile"
		buffer := bytes.Buffer{}
		throw(tmpl.Execute(&buffer, vars))
		minified, err := htmlmin.Minify(buffer.Bytes(), &htmlmin.Options{
			MinifyScripts: true,
		})
		throw(err)
		hasher := md5.New()
		hasher.Write(minified)
		Resources[kind] = Template{
			strings.Split(string(minified), "$$$"),
			hex.EncodeToString(hasher.Sum(nil))[:16],
		}
	}
}
