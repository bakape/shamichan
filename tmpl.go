/*
 Compiles HTML templates to be used for rendering pages during dynamic content
 insertion
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dchest/htmlmin"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// templateStore stores the static part of HTML templates and the corresponding
// truncated MD5 hash of said template
type templateStore struct {
	Parts [][]byte
	Hash  string
}

// templateMap stores all available templates
type templateMap map[string]templateStore

// resources exports temolates and their hashes by language
var resources templateMap

// compileTemplates reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func compileTemplates() {
	// Only one for now, but there will be more later
	raw := map[string]string{}
	for _, name := range []string{"index"} {
		file, err := ioutil.ReadFile("./tmpl/" + name + ".html")
		throw(err)
		raw[name] = string(file)
	}
	newResources := templateMap{}
	index, mobile := indexTemplate(raw["index"])
	newResources["index"] = index
	newResources["mobile"] = mobile
	resources = newResources
}

// clientFileHash is the combined, shortened MD5 hash of all client files
var clientFileHash string

type templateVars struct {
	Config                           template.JS
	Navigation                       template.HTML
	ClientHash, ConfigHash, MediaURL string
	IsMobile                         bool
}

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate(raw string) (templateStore, templateStore) {
	vars := templateVars{
		ConfigHash: configHash,
		MediaURL:   config.Hard.HTTP.Media,
	}
	js, err := json.Marshal(clientConfig)
	throw(err)
	vars.Config = template.JS(js)
	vars.Navigation = boardNavigation()
	hash := hashClientFiles()
	vars.ClientHash = hash
	clientFileHash = hash

	tmpl, err := template.New("index").Parse(raw)
	throw(err)

	// Rigt now the desktop and mobile templates are almost identical. This will
	// change, when we get a dedicated mobile GUI.
	return buildIndexTemplate(tmpl, vars, false),
		buildIndexTemplate(tmpl, vars, true)
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

	// Read all files into a buffer and hash it
	buf := new(bytes.Buffer)
	for _, file := range files {
		copyFile(file, buf)
	}
	return hashBuffer(buf.Bytes())
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
	files, err := dir.Readdirnames(0)
	throw(err)
	return files
}

// copyFile reads a file from disk and copies it into the writer
func copyFile(path string, writer io.Writer) {
	file, err := os.Open(path)
	throw(err)
	defer file.Close()
	_, err = io.Copy(writer, file)
	throw(err)
}

// boardNavigation renders interboard navigation we put in the top banner
func boardNavigation() template.HTML {
	html := `<b id="navTop">[`

	// Actual boards
	boards := config.Boards.Enabled
	for i, board := range boards {
		if board == config.Boards.Staff {
			continue
		}
		if i > 0 {
			html += ` / `
		}
		html += fmt.Sprintf(`<a href="../%v/" class="history">%v</a>`, board,
			board)
	}

	// Add custom URLs to board navigation
	for _, link := range config.Boards.Psuedo {
		html += fmt.Sprintf(` / <a href="%v">%v</a>`, link[1], link[0])
	}
	html += `]</b>`
	return template.HTML(html)
}

// buildIndexTemplate constructs the HTML template array, minifies and hashes it
func buildIndexTemplate(tmpl *template.Template, vars templateVars, isMobile bool) templateStore {
	vars.IsMobile = isMobile
	buffer := bytes.Buffer{}
	throw(tmpl.Execute(&buffer, vars))
	minified, err := htmlmin.Minify(buffer.Bytes(), &htmlmin.Options{
		MinifyScripts: true,
	})
	throw(err)
	return templateStore{
		bytes.Split(minified, []byte("<$$$>")),
		hashBuffer(minified),
	}
}
