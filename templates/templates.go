// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/dchest/htmlmin"
	"html/template"
	"path/filepath"
)

// Overriden in tests
var templateRoot = "templates"

// Store stores the compiled HTML template and the corresponding truncated MD5
// hash of said template
type Store struct {
	HTML []byte
	Hash string
}

// Map stores all available templates
type Map map[string]Store

// Resources conatains all available templates
var Resources = Map{}

// Compile reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func Compile() error {
	// Only one for now, but there will be more later
	index, mobile, err := indexTemplate()
	if err != nil {
		return err
	}
	Resources["index"] = index
	Resources["mobile"] = mobile
	return nil
}

// clientFileHash is the combined, shortened MD5 hash of all client files
var clientFileHash string

type vars struct {
	Config     template.JS
	Navigation template.HTML
	ConfigHash string
	IsMobile   bool
}

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate() (desktop Store, mobile Store, err error) {
	v := vars{ConfigHash: config.Hash}
	v.Config = template.JS(config.ClientConfig)
	v.Navigation = boardNavigation()
	tmpl, err := template.ParseFiles(
		filepath.FromSlash(templateRoot + "/index.html"),
	)
	if err != nil {
		err = util.WrapError("Error parsing index temlate", err)
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
	conf := config.Config.Boards

	// Actual boards and "/all/" metaboard
	for i, board := range append(conf.Enabled, "all") {
		if board == conf.Staff {
			continue
		}
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
func buildIndexTemplate(
	tmpl *template.Template,
	vars vars,
	isMobile bool,
) (Store, error) {
	vars.IsMobile = isMobile
	buffer := new(bytes.Buffer)
	if err := tmpl.Execute(buffer, vars); err != nil {
		return Store{}, util.WrapError("Error compiling index template", err)
	}
	opts := &htmlmin.Options{MinifyScripts: true}
	minified, err := htmlmin.Minify(buffer.Bytes(), opts)
	if err != nil {
		return Store{}, util.WrapError("Error minifying index template", err)
	}
	hash, err := util.HashBuffer(minified)
	if err != nil {
		return Store{}, util.WrapError("Error hashing index template", err)
	}
	return Store{minified, hash}, nil
}
