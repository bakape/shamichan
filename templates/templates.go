// Package templates generates and stores HTML templates
package templates

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/lang"
	"github.com/bakape/meguca/util"
)

var (
	// TemplateRoot stores the root directory of all HTML templates. Overridden
	// in tests.
	TemplateRoot = "templates"

	// resources contains all available templates
	resources = make(map[string]Store, 2)

	// clientFileHash is the combined, shortened MD5 hash of all client files
	clientFileHash string

	mu sync.RWMutex

	// Contains all compiled HTML templates
	tmpl = make(map[string]*template.Template)

	// Template functions for rendering posts
	postFunctions = template.FuncMap{
		"thumbPath":        thumbPath,
		"renderTime":       renderTime,
		"readableLength":   readableLength,
		"readableFileSize": readableFileSize,
		"sourcePath":       sourcePath,
		"extension":        extension,
		"wrapPost":         wrapPost,
		"renderPostLink":   renderPostLink,
		"renderBody":       renderBody,
	}

	isTest bool
)

// Store stores the compiled HTML template and the corresponding truncated MD5
// hash of said template
type Store struct {
	HTML []byte
	Hash string
}

// Template variables
type vars struct {
	Captcha                       bool
	Config                        template.JS
	Email, ConfigHash, DefaultCSS string
	FAQ                           template.HTML
	Lang                          lang.Pack
	Identity, Login, Register     []inputSpec
	Options                       [][]inputSpec
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
		{"captcha", nil, nil},
		{"threadForm", []string{"captcha"}, nil},
		{"article", nil, postFunctions},
		{"index", nil, template.FuncMap{
			"table":  renderTable,
			"bundle": bundle,
			"input":  renderInput,
			"label":  renderLabel,
		}},
		{"noscript", nil, nil},
		{
			"board",
			[]string{"threadForm", "captcha"},
			template.FuncMap{
				"thumbPath": thumbPath,
			},
		},
		{"thread", []string{"article"}, postFunctions},
	}

	for _, s := range specs {
		path := filepath.Join(TemplateRoot, s.name+".html")
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// Strip tabs and newlines. Not done in tests to preserve HTML line
		// numbers in template errors.
		var min []byte
		if !isTest {
			min = make([]byte, 0, len(b))
			for _, r := range b {
				if r != '\n' && r != '\t' {
					min = append(min, r)
				}
			}
		} else {
			min = b
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
	index, err := buildIndexTemplate()
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()
	resources["index"] = index
	return nil
}

// buildIndexTemplate constructs the HTML template array, minifies and hashes it
func buildIndexTemplate() (store Store, err error) {
	clientJSON, hash := config.GetClient()
	conf := config.Get()

	v := vars{
		Config:     template.JS(clientJSON),
		ConfigHash: hash,
		Captcha:    conf.Captcha,
		Email:      conf.FeedbackEmail,
		DefaultCSS: conf.DefaultCSS,
		FAQ:        template.HTML(strings.Replace(conf.FAQ, "\n", "<br>", -1)),
		Boards:     config.GetBoards(),
		ImageSearch: []imageSearch{
			{"google", "G"},
			{"iqdb", "Iq"},
			{"saucenao", "Sn"},
			{"desustorage", "Ds"},
			{"exhentai", "Ex"},
		},
		SortModes: []string{
			"lastReply", "creation", "replyCount", "fileCount",
		},
		Identity: specs["identity"],
		Login:    specs["login"],
		Register: specs["register"],
		Options:  optionSpecs,
		Lang:     lang.Packs["en_GB"],
	}

	w := new(bytes.Buffer)
	err = tmpl["index"].Execute(w, v)
	if err != nil {
		return
	}
	buf := w.Bytes()

	return Store{buf, util.HashBuffer(buf)}, nil
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

// Bundles several values for passing down template pipelines together
func bundle(vals ...interface{}) []interface{} {
	return vals
}
