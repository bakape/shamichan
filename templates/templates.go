// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/util"
	"github.com/dchest/htmlmin"
)

var (
	// TemplateRoot stores the root directory of all HTML templates. Overridden
	// in tests.
	TemplateRoot = "templates"

	// resources contains all available templates
	resources = make(map[string]Store, 2)

	// clientFileHash is the combined, shortened MD5 hash of all client files
	clientFileHash string

	imageSearchEngines = []imageSearch{
		{"google", "G"},
		{"iqdb", "Iq"},
		{"saucenao", "Sn"},
		{"desustorage", "Ds"},
		{"exhentai", "Ex"},
	}

	sortModes = []string{"lastReply", "creation", "replyCount", "fileCount"}

	mu sync.RWMutex

	// Contains all compiled HTML templates
	tmpl = make(map[string]*template.Template)
)

// Store stores the compiled HTML template and the corresponding truncated MD5
// hash of said template
type Store struct {
	HTML []byte
	Hash string
}

// Template variables
type vars struct {
	IsMobile, Captcha             bool
	Config                        template.JS
	Email, ConfigHash, DefaultCSS string
	ImageSearch                   []imageSearch
	SortModes, Boards             []string
}

// Definition for an image search link
type imageSearch struct {
	ID, Abbrev string
}

// ParseTemplates reads all HTML templates from disk, strips whitespace and
// parses them into the global template map
func ParseTemplates() error {
	specs := [...]struct {
		name string
		deps []string
		fns  template.FuncMap
	}{
		// Order matters. Dependencies must come before dependents.
		{"index", nil, nil},
		{"noscript", nil, nil},
		{"board", nil, template.FuncMap{
			"thumbPath": thumbPath,
		}},
		{"thread", nil, template.FuncMap{
			"thumbPath": thumbPath,
		}},
	}

	for _, s := range specs {
		path := filepath.Join(TemplateRoot, s.name+".html")
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// Strip tabs and newlines
		min := make([]byte, 0, len(b))
		for _, r := range b {
			if r != '\n' && r != '\t' {
				min = append(min, r)
			}
		}

		t := template.New(s.name).Funcs(s.fns)
		for _, d := range s.deps {
			_, err := t.AddParseTree(d, tmpl[d].Tree)
			if err != nil {
				return err
			}
		}
		t, err = t.Parse(string(min))
		if err != nil {
			return err
		}
		tmpl[s.name] = t
	}

	return nil
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

// indexTemplate compiles the HTML template for thread and board pages of the
// imageboard
func indexTemplate() (desktop Store, mobile Store, err error) {
	clientJSON, hash := config.GetClient()
	conf := config.Get()

	v := vars{
		Config:      template.JS(clientJSON),
		ConfigHash:  hash,
		Captcha:     conf.Captcha,
		Email:       conf.FeedbackEmail,
		DefaultCSS:  conf.DefaultCSS,
		ImageSearch: imageSearchEngines,
		SortModes:   sortModes,
		Boards:      config.GetBoards(),
	}

	// Right now the desktop and mobile templates are almost identical. This
	// will change, when we get a dedicated mobile GUI.
	desktop, err = buildIndexTemplate(v, false)
	if err != nil {
		return
	}
	mobile, err = buildIndexTemplate(v, true)
	return
}

// buildIndexTemplate constructs the HTML template array, minifies and hashes it
func buildIndexTemplate(vars vars, isMobile bool) (store Store, err error) {
	vars.IsMobile = isMobile
	buffer := new(bytes.Buffer)
	err = tmpl["index"].Execute(buffer, vars)
	if err != nil {
		return
	}

	minified, err := htmlmin.Minify(buffer.Bytes(), &htmlmin.Options{
		MinifyScripts: true,
	})
	if err != nil {
		return
	}

	// Also strip all newlines
	minified = bytes.Replace(minified, []byte{'\n'}, []byte{}, -1)

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
