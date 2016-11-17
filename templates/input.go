// Form input field rendering

package templates

import (
	"bytes"
	"html"
	"html/template"
	"strconv"

	"strings"

	"github.com/bakape/meguca/lang"
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
	typ                         inputType
	required, placeholder, noID bool
	min, max, maxLength, rows   int
	id, pattern                 string
	options                     []string
	val                         interface{}
}

type htmlWriter struct {
	bytes.Buffer
	lang lang.Pack
}

// Write an element attribute to the buffer
func (w *htmlWriter) attr(key, val string) {
	w.WriteByte(' ')
	w.WriteString(key)
	if val != "" {
		w.WriteString(`="`)
		w.WriteString(val)
		w.WriteByte('"')
	}
}

func (w *htmlWriter) typ(val string) {
	w.attr("type", val)
}

// Write an input element from the spec to the buffer
func (w *htmlWriter) input(spec inputSpec) error {
	cont := false
	switch spec.typ {
	case _select:
		w.sel(spec)
	case _textarea:
		w.textArea(spec)
	case _map:
		return w.writeMap(spec)
	case _shortcut:
		w.WriteString("Alt+")
		cont = true
	default:
		cont = true
	}
	if !cont {
		return nil
	}

	w.tag("input", spec)

	switch spec.typ {
	case _bool:
		w.typ("checkbox")
		if spec.val != nil && spec.val.(bool) {
			w.attr("checked", "")
		}
	case _number:
		w.typ("number")
		if spec.val != nil {
			w.attr("value", strconv.Itoa(spec.val.(int)))
		}
		w.attr("min", strconv.Itoa(spec.min))
		if spec.max != 0 {
			w.attr("max", strconv.Itoa(spec.max))
		}
	case _password, _string:
		if spec.typ == _string {
			w.typ("text")
		} else {
			w.typ("password")
		}
		if spec.val != nil {
			w.attr("value", spec.val.(string))
		}
		if spec.pattern != "" {
			w.attr("pattern", spec.pattern)
		}
		if spec.maxLength != 0 {
			w.attr("maxlength", strconv.Itoa(spec.maxLength))
		}
	case _image:
		w.typ("file")
		w.attr("accept", "image/png,image/gif,image/jpeg")
	case _shortcut:
		w.attr("maxlength", "1")
		w.attr("class", "shortcut")
	}

	w.WriteByte('>')
	return nil
}

// Write the element tag and the common parts of all input element types to
// buffer
func (w *htmlWriter) tag(tag string, spec inputSpec) {
	w.WriteByte('<')
	w.WriteString(tag)
	w.attr("name", spec.id)
	if !spec.noID { // To not conflict with non-unique labels
		w.attr("id", spec.id)
	}
	w.attr("title", w.lang.Forms[spec.id][1])
	if spec.placeholder {
		w.attr("placeholder", w.lang.Forms[spec.id][0])
	}
	if spec.required {
		w.attr("required", "")
	}
}

// Write an HTML-escaped string to buffer
func (w *htmlWriter) escape(s string) {
	w.WriteString(html.EscapeString(s))
}

// Write a select element to buffer
func (w *htmlWriter) sel(spec inputSpec) {
	w.tag("select", spec)
	w.WriteByte('>')

	var val string
	if spec.val != nil {
		val = spec.val.(string)
	}

	for _, o := range spec.options {
		w.WriteString("<option")
		w.attr("value", o)
		if o == val {
			w.attr("selected", "selected")
		}
		w.WriteByte('>')

		label, ok := w.lang.Options[spec.id]
		if !ok {
			label = o
		}
		w.WriteString(label)

		w.WriteString("</option>")
	}

	w.WriteString("</select>")
}

// Render a text area input element
func (w *htmlWriter) textArea(spec inputSpec) {
	w.tag("textarea", spec)
	if spec.maxLength != 0 {
		w.attr("maxlength", strconv.Itoa(spec.maxLength))
	}
	if spec.rows == 0 {
		spec.rows = 3
	}
	w.attr("rows", strconv.Itoa(spec.rows))
	w.WriteByte('>')

	switch spec.val.(type) {
	case string:
		w.escape(spec.val.(string))
	case []string:
		w.escape(strings.Join(spec.val.([]string), "\n"))
	}

	w.WriteString("</textarea>")
}

// Write a subform for inputting a key-value string map to buffer
func (w *htmlWriter) writeMap(spec inputSpec) error {
	return tmpl["map"].Execute(w, struct {
		Spec inputSpec
		Lang lang.Pack
	}{
		Spec: spec,
		Lang: w.lang,
	})
}

// Write an input element label from the spec to the buffer
func (w *htmlWriter) label(spec inputSpec) {
	ln := w.lang.Forms[spec.id]

	w.WriteString("<label")
	if !spec.noID {
		w.attr("for", spec.id)
	}
	w.attr("title", ln[1])
	w.WriteByte('>')

	w.WriteString(ln[0])
	w.WriteString("</label>")
}

// Render a table containing {label input_element} pairs
func renderTable(specs []inputSpec, lang lang.Pack) template.HTML {
	w := htmlWriter{
		lang: lang,
	}
	w.WriteString("<table>")

	for _, spec := range specs {
		w.WriteString("<tr><td>")
		w.label(spec)
		w.WriteString("</td><td>")
		w.input(spec)
		w.WriteString("</td></tr>")
	}

	w.WriteString("</table>")

	return template.HTML(w.String())
}

// Render a single input element
func renderInput(spec inputSpec, lang lang.Pack) (template.HTML, error) {
	w := htmlWriter{
		lang: lang,
	}
	if err := w.input(spec); err != nil {
		return template.HTML(""), err
	}
	return template.HTML(w.String()), nil
}

// Render a single label for an input element
func renderLabel(spec inputSpec, lang lang.Pack) template.HTML {
	w := htmlWriter{
		lang: lang,
	}
	w.label(spec)
	return template.HTML(w.String())
}
