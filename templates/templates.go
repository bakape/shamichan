// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"html/template"
	"path/filepath"
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/dchest/htmlmin"
)

var (
	// TemplateRoot stores the root directory of all HTML templates. Overriden
	// in tests.
	TemplateRoot = "templates"

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
	Captcha    bool
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
		Captcha:    conf.Captcha,
		Email:      conf.FeedbackEmail,
		DefaultCSS: conf.DefaultCSS,
	}
	path := filepath.FromSlash(TemplateRoot + "/index.html")
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

	minified, err := htmlmin.Minify(buffer.Bytes(), &htmlmin.Options{
		MinifyScripts: true,
	})
	if err != nil {
		return
	}

	return Store{minified, util.HashBuffer(minified)}, nil
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
