// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"fmt"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/dchest/htmlmin"
	"html/template"
)

// Overriden in tests
var templateRoot = "./templates"

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
func Compile() {
	// Only one for now, but there will be more later
	index, mobile := indexTemplate()
	Resources["index"] = index
	Resources["mobile"] = mobile
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
func indexTemplate() (Store, Store) {
	v := vars{ConfigHash: config.Hash}
	v.Config = template.JS(config.ClientConfig)
	v.Navigation = boardNavigation()
	tmpl, err := template.ParseFiles(templateRoot + "/index.html")
	util.Throw(err)

	// Rigt now the desktop and mobile templates are almost identical. This will
	// change, when we get a dedicated mobile GUI.
	return buildIndexTemplate(tmpl, v, false),
		buildIndexTemplate(tmpl, v, true)
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
) Store {
	vars.IsMobile = isMobile
	buffer := new(bytes.Buffer)
	util.Throw(tmpl.Execute(buffer, vars))
	minified, err := htmlmin.Minify(buffer.Bytes(), &htmlmin.Options{
		MinifyScripts: true,
	})
	util.Throw(err)
	return Store{
		minified,
		util.HashBuffer(minified),
	}
}
