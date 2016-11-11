// Form input field rendering

package templates

import (
	"bytes"
	"html/template"
	"strconv"

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
func (w *htmlWriter) writeAttr(key, val string) {
	w.WriteByte(' ')
	w.WriteString(key)
	if val != "" {
		w.WriteString(`="`)
		w.WriteString(val)
		w.WriteByte('"')
	}
}

func (w *htmlWriter) writeType(val string) {
	w.writeAttr("type", val)
}

// Write an input element from the spec to the buffer
func (w *htmlWriter) writeInput(spec inputSpec) {
	cont := false
	switch spec.typ {
	case _select:
		w.writeSelect(spec)
	case _shortcut:
		w.WriteString("Alt+")
		cont = true
	default:
		cont = true
	}
	if !cont {
		return
	}

	w.writeTag("input", spec)

	switch spec.typ {
	case _bool:
		w.writeType("checkbox")
		if spec.val != nil {
			w.writeAttr("checked", "")
		}
	case _number:
		w.writeType("number")
		if spec.val != nil {
			w.writeAttr("value", strconv.Itoa(spec.val.(int)))
		}
		if spec.min != 0 {
			// Zero is the nil value, so when we actually actually want "0",
			// pass -1
			if spec.min == -1 {
				spec.min = 0
			}
			w.writeAttr("min", strconv.Itoa(spec.min))
		}
		if spec.max != 0 {
			w.writeAttr("max", strconv.Itoa(spec.max))
		}
	case _password, _string:
		if spec.typ == _string {
			w.writeType("text")
		} else {
			w.writeType("password")
		}
		if spec.val != nil {
			w.writeAttr("value", spec.val.(string))
		}
		if spec.pattern != "" {
			w.writeAttr("pattern", spec.pattern)
		}
		if spec.maxLength != 0 {
			w.writeAttr("maxlength", strconv.Itoa(spec.maxLength))
		}
	case _image:
		w.writeType("file")
		w.writeAttr("accept", "image/png,image/gif,image/jpeg")
	case _shortcut:
		w.writeAttr("maxlength", "1")
		w.writeAttr("class", "shortcut")
	}

	w.WriteByte('>')
}

// Write the element tag and the common parts of all input element types to
// buffer
func (w *htmlWriter) writeTag(tag string, spec inputSpec) {
	w.WriteByte('<')
	w.WriteString(tag)
	w.writeAttr("name", spec.id)
	if !spec.noID { // To not conflict with non-unique labels
		w.writeAttr("id", spec.id)
	}
	w.writeAttr("title", w.lang.Forms[spec.id][1])
	if spec.placeholder {
		w.writeAttr("placeholder", w.lang.Forms[spec.id][0])
	}
	if spec.required {
		w.writeAttr("required", "")
	}
}

// Write a select element to buffer
func (w *htmlWriter) writeSelect(spec inputSpec) {
	w.writeTag("select", spec)
	w.WriteByte('>')

	var val string
	if spec.val != nil {
		val = spec.val.(string)
	}

	for _, o := range spec.options {
		w.WriteString("<option")
		w.writeAttr("value", spec.id)
		if o == val {
			w.writeAttr("selected", "selected")
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

// Write an input element label from the spec to the buffer
func (w *htmlWriter) writeLabel(spec inputSpec) {
	ln := w.lang.Forms[spec.id]

	w.WriteString("<label")
	if !spec.noID {
		w.writeAttr("for", spec.id)
	}
	w.writeAttr("title", ln[1])
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
		w.writeLabel(spec)
		w.WriteString("</td><td>")
		w.writeInput(spec)
		w.WriteString("</td></tr>")
	}

	w.WriteString("</table>")

	return template.HTML(w.String())
}

// Render a single input element
func renderInput(spec inputSpec, lang lang.Pack) template.HTML {
	w := htmlWriter{
		lang: lang,
	}
	w.writeInput(spec)
	return template.HTML(w.String())
}

// Render a single label for an input element
func renderLabel(spec inputSpec, lang lang.Pack) template.HTML {
	w := htmlWriter{
		lang: lang,
	}
	w.writeLabel(spec)
	return template.HTML(w.String())
}
