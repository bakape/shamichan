// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/tdewolff/minify"
	htmlMin "github.com/tdewolff/minify/html"
)

var (
	// Overriden in tests
	templateRoot = "templates"

	// resources conatains all available templates
	resources = map[string]Store{}

	mu sync.RWMutex
)

// Store stores the compiled HTML template and the corresponding truncated MD5
// hash of said template
type Store struct {
	HTML []byte
	Hash string
}

// Compile reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func Compile() error {
	// Only one for now, but there will be more later
	index, mobile, err := indexTemplate()
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()
	resources["index"] = index
	resources["mobile"] = mobile
	return nil
}

// clientFileHash is the combined, shortened MD5 hash of all client files
var clientFileHash string

type vars struct {
	IsMobile   bool
	Config     template.JS
	Navigation template.HTML
	Email      string
	ConfigHash string
	DefaultCSS string
}

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate() (desktop Store, mobile Store, err error) {
	clientJSON, hash := config.GetClient()
	conf := config.Get()
	v := vars{
		Config:     template.JS(clientJSON),
		ConfigHash: hash,
		Navigation: boardNavigation(),
		Email:      conf.FeedbackEmail,
		DefaultCSS: conf.DefaultCSS,
	}
	path := filepath.FromSlash(templateRoot + "/index.html")
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		err = util.WrapError("error parsing index temlate", err)
		return
	}

	// Rigt now the desktop and mobile templates are almost identical. This will
	// change, when we get a dedicated mobile GUI.
	desktop, err = buildIndexTemplate(tmpl, v, false)
	if err != nil {
		return
	}
	mobile, err = buildIndexTemplate(tmpl, v, true)
	return
}

// boardNavigation renders interboard navigation we put in the top banner
func boardNavigation() template.HTML {
	html := `<b id="navTop">[`
	conf := config.Get().Boards

	// Actual boards and "/all/" metaboard
	for i, board := range append(conf.Enabled, "all") {
		html += boardLink(i > 0, board, "../"+board+"/")
	}

	// Add custom URLs to board navigation
	for _, link := range conf.Psuedo {
		html += boardLink(true, link[0], link[1])
	}
	html += `]</b>`
	return template.HTML(html)
}

// Builds a a board link, for the interboard navigation bar
func boardLink(notFirst bool, name, url string) string {
	link := fmt.Sprintf(`<a href="%v">%v</a>`, url, name)
	if notFirst {
		link = " / " + link
	}
	return link
}

// buildIndexTemplate constructs the HTML template array, minifies and hashes it
func buildIndexTemplate(tmpl *template.Template, vars vars, isMobile bool) (
	store Store, err error,
) {
	vars.IsMobile = isMobile
	buffer := new(bytes.Buffer)
	err = tmpl.Execute(buffer, vars)
	if err != nil {
		return
	}

	min := minify.New()
	min.AddFunc("text/html", htmlMin.Minify)
	minified, err := min.Bytes("text/html", buffer.Bytes())
	if err != nil {
		return
	}

	hash, err := util.HashBuffer(minified)
	if err != nil {
		return
	}
	return Store{minified, hash}, nil
}

// Get retrieves a compiled template by its name
func Get(name string) Store {
	mu.RLock()
	defer mu.RUnlock()
	return resources[name]
}

// Set sets a template to the specified value. Only use in tests.
func Set(name string, s Store) {
	mu.Lock()
	defer mu.Unlock()
	resources[name] = s
}
