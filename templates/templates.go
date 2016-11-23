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
)

var (
	// TemplateRoot stores the root directory of all HTML templates. Overridden
	// in tests.
	TemplateRoot = "templates"

	// resources contains compiled index templates for every language
	indexTemplates = make(map[string]*template.Template, 6)

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

// Variables for the second pass template execution
type secondPassVars struct {
	Title   string
	Threads template.HTML
}

// Definition for an image search link
type imageSearch struct {
	ID, Abbrev string
}

// Parse reads all HTML templates from disk, strips whitespace and parses them
// into the global template map
func Parse() error {
	specs := [...]struct {
		id   string
		deps []string
		fns  template.FuncMap
	}{
		// Order matters. Dependencies must come before dependents.
		{id: "captcha"},
		{id: "hover-reveal"},
		{id: "boardNavigation"},
		{id: "ownedBoard"},
		{
			"createBoard",
			[]string{"captcha"},
			template.FuncMap{
				"bundle": bundle,
				"table":  renderTable,
			},
		},
		{
			"tableForm",
			nil,
			template.FuncMap{
				"table": renderTable,
			},
		},
		{id: "keyValue"},
		{
			"map",
			[]string{"keyValue"},
			template.FuncMap{
				"bundle": bundle,
			},
		},
		{"article", nil, postFunctions},
		{
			"index",
			[]string{"captcha", "keyValue"},
			template.FuncMap{
				"table":  renderTable,
				"bundle": bundle,
				"input":  renderInput,
				"label":  renderLabel,
			},
		},
		{
			"board",
			[]string{"captcha", "hover-reveal"},
			template.FuncMap{
				"thumbPath": thumbPath,
				"bundle":    bundle,
			},
		},
		{
			"thread",
			[]string{"article", "captcha"},
			postFunctions,
		},
	}

	for _, s := range specs {
		path := filepath.Join(TemplateRoot, "html", s.id+".html")
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

		t := template.New(s.id).Funcs(s.fns)
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
		tmpl[s.id] = t
	}

	return nil
}

// Compile reads template HTML from disk, injects dynamic variables,
// hashes and stores them
func Compile() (err error) {
	t := make(map[string]*template.Template, len(lang.Packs))
	for id := range lang.Packs {
		t[id], err = buildIndexTemplate(lang.Packs[id])
		if err != nil {
			return err
		}
	}

	mu.Lock()
	indexTemplates = t
	mu.Unlock()

	return nil
}

// buildIndexTemplate constructs the HTML template array, minifies and hashes it
func buildIndexTemplate(ln lang.Pack) (*template.Template, error) {
	clientJSON, hash := config.GetClient()
	conf := config.Get()

	v := struct {
		Captcha                       bool
		Config                        template.JS
		Email, ConfigHash, DefaultCSS string
		FAQ                           template.HTML
		Lang                          lang.Pack
		Identity, Login, Register     []inputSpec
		Options                       [][]inputSpec
		ImageSearch                   []imageSearch
		Boards                        []string
	}{
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
		Identity: specs["identity"],
		Login:    specs["login"],
		Register: specs["register"],
		Options:  optionSpecs,
		Lang:     ln,
	}

	w := new(bytes.Buffer)
	err := tmpl["index"].Execute(w, v)
	if err != nil {
		return nil, err
	}

	// Second template compile pass
	firstPass := w.String()
	w.Reset()
	t := template.New(ln.ID)
	t, err = t.Parse(firstPass)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Bundles several values for passing down template pipelines together
func bundle(vals ...interface{}) []interface{} {
	return vals
}
