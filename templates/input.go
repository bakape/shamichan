// Form input field rendering

package templates

import (
	"html"
	"strconv"
	"strings"

	"../lang"
	"github.com/valyala/quicktemplate"
)

// Input field types
type inputType uint8

const (
	_bool inputType = iota
	_number
	_string
	_select
	_textarea
	_password
	_map
	_image
	_shortcut
)

// Spec of an option passed into the rendering function
type inputSpec struct {
	Type                        inputType
	Required, Placeholder, NoID bool
	Min, Max, MaxLength, Rows   int
	ID, Pattern, Autocomplete   string
	Options                     []string
	Val                         interface{}
}

// For constructing various HTML input forms
type formWriter struct {
	quicktemplate.Writer
	lang lang.Pack
}

// Write an element attribute to the buffer
func (w *formWriter) attr(key, val string) {
	w.N().S(` `)
	w.N().S(key)
	if val != "" {
		w.N().S(`="`)
		w.N().S(val)
		w.N().S(`"`)
	}
}

func (w *formWriter) typ(val string) {
	w.attr("type", val)
}

// Write an input element from the spec to the buffer
func (w *formWriter) input(spec inputSpec) {
	cont := false
	switch spec.Type {
	case _select:
		w.sel(spec)
	case _textarea:
		w.textArea(spec)
	case _map:
		streamrenderMap(&w.Writer, spec, w.lang)
	case _shortcut:
		w.N().S("Alt+")
		cont = true
	default:
		cont = true
	}
	if !cont {
		return
	}

	w.tag("input", spec)

	switch spec.Type {
	case _bool:
		w.typ("checkbox")
		if spec.Val != nil && spec.Val.(bool) {
			w.attr("checked", "")
		}
	case _number:
		w.typ("number")
		if spec.Val != nil {
			cast := uint64(spec.Val.(uint))
			w.attr("value", strconv.FormatUint(cast, 10))
		}
		w.attr("min", strconv.Itoa(spec.Min))
		if spec.Max != 0 {
			w.attr("max", strconv.Itoa(spec.Max))
		}
	case _password, _string:
		if spec.Type == _string {
			w.typ("text")
		} else {
			w.typ("password")
		}
		if spec.Val != nil {
			w.attr("value", html.EscapeString(spec.Val.(string)))
		}
		if spec.Pattern != "" {
			w.attr("pattern", spec.Pattern)
		}
		if spec.MaxLength != 0 {
			w.attr("maxlength", strconv.Itoa(spec.MaxLength))
		}
		if spec.Autocomplete != "" {
			w.attr("autocomplete", spec.Autocomplete)
		}
	case _image:
		w.typ("file")
		w.attr("accept", "image/png,image/gif,image/jpeg")
	case _shortcut:
		w.attr("maxlength", "1")
		w.attr("class", "shortcut")
	}

	w.N().S(`>`)
}

// Write the element tag and the common parts of all input element types to
// buffer
func (w *formWriter) tag(tag string, spec inputSpec) {
	w.N().S(`<`)
	w.N().S(tag)
	w.attr("name", spec.ID)
	if !spec.NoID { // To not conflict with non-unique labels
		w.attr("id", spec.ID)
	}
	w.attr("title", w.lang.Forms[spec.ID][1])
	if spec.Placeholder {
		w.attr("placeholder", w.lang.Forms[spec.ID][0])
	}
	if spec.Required {
		w.attr("required", "")
	}
}

// Write a select element to buffer
func (w *formWriter) sel(spec inputSpec) {
	w.tag("select", spec)
	w.N().S(`>`)

	var val string
	if spec.Val != nil {
		val = spec.Val.(string)
	}

	for _, o := range spec.Options {
		w.N().S("<option")
		w.attr("value", o)
		if o == val {
			w.attr("selected", "selected")
		}
		w.N().S(`>`)

		label, ok := w.lang.Options[spec.ID]
		if !ok {
			label = o
		}
		w.N().S(label)

		w.N().S("</option>")
	}

	w.N().S("</select>")
}

// Render a text area input element
func (w *formWriter) textArea(spec inputSpec) {
	w.tag("textarea", spec)
	if spec.MaxLength != 0 {
		w.attr("maxlength", strconv.Itoa(spec.MaxLength))
	}
	w.attr("rows", strconv.Itoa(spec.Rows))
	w.N().S(`>`)

	switch spec.Val.(type) {
	case string:
		w.E().S(spec.Val.(string))
	case []string:
		w.E().S(strings.Join(spec.Val.([]string), "\n"))
	}

	w.N().S("</textarea>")
}

// Write an input element label from the spec to the buffer
func (w *formWriter) label(spec inputSpec) {
	ln := w.lang.Forms[spec.ID]

	w.N().S("<label")
	if !spec.NoID {
		w.attr("for", spec.ID)
	}
	w.attr("title", ln[1])
	w.N().S(`>`)

	w.N().S(ln[0])
	w.N().S("</label>")
}

// Render a table containing {label input_element} pairs
func streamtable(qw *quicktemplate.Writer, specs []inputSpec, lang lang.Pack) {
	w := formWriter{
		Writer: *qw,
		lang:   lang,
	}
	w.N().S("<table>")

	for _, spec := range specs {
		w.N().S("<tr><td>")
		w.label(spec)
		w.N().S("</td><td>")
		w.input(spec)
		w.N().S("</td></tr>")
	}

	w.N().S("</table>")
}

// Render a single input element
func streaminput(qw *quicktemplate.Writer, spec inputSpec, lang lang.Pack) {
	w := formWriter{
		Writer: *qw,
		lang:   lang,
	}
	w.input(spec)
}

// Render the options inputs of an options panel
func streamoptions(qw *quicktemplate.Writer, specs []inputSpec, ln lang.Pack) {
	w := formWriter{
		Writer: *qw,
		lang:   ln,
	}
	for _, s := range specs {
		w.input(s)
		w.label(s)
		w.N().S(`<br>`)
	}
}
