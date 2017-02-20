package templates

import "bytes"

type tokenType uint8

const (
	unmatched tokenType = iota
	word
	quoted
	doubleQuoted
	comment
)

const (
	operatorHeader = "<span class=\"ms-operator\">"
	functionHeader = "<span class=\"ms-function\">"
	stringHeader   = "<span class=\"ms-string\">"
	commentHeader  = "<span class=\"ms-comment\">"
)

var keywords = map[string]bool{
	"NULL":         true,
	"NaN":          true,
	"abstract":     true,
	"alias":        true,
	"and":          true,
	"arguments":    true,
	"array":        true,
	"asm":          true,
	"assert":       true,
	"async":        true,
	"auto":         true,
	"await":        true,
	"base":         true,
	"begin":        true,
	"bool":         true,
	"boolean":      true,
	"break":        true,
	"byte":         true,
	"case":         true,
	"catch":        true,
	"char":         true,
	"checked":      true,
	"class":        true,
	"clone":        true,
	"compl":        true,
	"const":        true,
	"continue":     true,
	"debugger":     true,
	"decimal":      true,
	"declare":      true,
	"default":      true,
	"defer":        true,
	"deinit":       true,
	"delegate":     true,
	"delete":       true,
	"do":           true,
	"double":       true,
	"echo":         true,
	"elif":         true,
	"else":         true,
	"elseif":       true,
	"elsif":        true,
	"end":          true,
	"ensure":       true,
	"enum":         true,
	"event":        true,
	"except":       true,
	"exec":         true,
	"explicit":     true,
	"export":       true,
	"extends":      true,
	"extension":    true,
	"extern":       true,
	"fallthrough":  true,
	"false":        true,
	"final":        true,
	"finally":      true,
	"fixed":        true,
	"float":        true,
	"fn":           true,
	"for":          true,
	"foreach":      true,
	"friend":       true,
	"from":         true,
	"func":         true,
	"function":     true,
	"global":       true,
	"go":           true,
	"goto":         true,
	"guard":        true,
	"if":           true,
	"impl":         true,
	"implements":   true,
	"implicit":     true,
	"import":       true,
	"in":           true,
	"int":          true,
	"include":      true,
	"inline":       true,
	"inout":        true,
	"instanceof":   true,
	"interface":    true,
	"internal":     true,
	"is":           true,
	"lambda":       true,
	"let":          true,
	"lock":         true,
	"long":         true,
	"module":       true,
	"mut":          true,
	"mutable":      true,
	"namespace":    true,
	"native":       true,
	"new":          true,
	"next":         true,
	"nil":          true,
	"not":          true,
	"null":         true,
	"object":       true,
	"operator":     true,
	"or":           true,
	"out":          true,
	"override":     true,
	"package":      true,
	"params":       true,
	"private":      true,
	"protected":    true,
	"protocol":     true,
	"pub":          true,
	"public":       true,
	"raise":        true,
	"readonly":     true,
	"redo":         true,
	"ref":          true,
	"register":     true,
	"repeat":       true,
	"require":      true,
	"rescue":       true,
	"restrict":     true,
	"retry":        true,
	"return":       true,
	"sbyte":        true,
	"sealed":       true,
	"short":        true,
	"signed":       true,
	"sizeof":       true,
	"static":       true,
	"str":          true,
	"string":       true,
	"struct":       true,
	"subscript":    true,
	"super":        true,
	"switch":       true,
	"synchronized": true,
	"template":     true,
	"then":         true,
	"throws":       true,
	"transient":    true,
	"true":         true,
	"try":          true,
	"type":         true,
	"typealias":    true,
	"typedef":      true,
	"typeid":       true,
	"typename":     true,
	"typeof":       true,
	"uint":         true,
	"unchecked":    true,
	"undef":        true,
	"undefined":    true,
	"union":        true,
	"unless":       true,
	"unsigned":     true,
	"until":        true,
	"use":          true,
	"using":        true,
	"var":          true,
	"virtual":      true,
	"void":         true,
	"volatile":     true,
	"when":         true,
	"where":        true,
	"while":        true,
	"with":         true,
	"xor":          true,
	"yield":        true,
}

var operators = map[byte]bool{
	'+': true,
	'-': true,
	'~': true,
	'!': true,
	'@': true,
	'%': true,
	'^': true,
	'&': true,
	'*': true,
	'=': true,
	'|': true,
	':': true,
	'<': true,
	'>': true,
	'?': true,
	'/': true,
}

type codeWriter struct {
	bytes.Buffer
}

func (w *codeWriter) escape(buf []byte) {
	for _, b := range buf {
		w.escapeByte(b)
	}
}

func (w *codeWriter) escapeByte(b byte) {
	var s string
	switch b {
	case '&':
		s = "&amp;"
	case '<':
		s = "&lt;"
	case '>':
		s = "&gt;"
	case '\'':
		s = "&#39;"
	case '"':
		s = "&#34;"
	}
	if s != "" {
		w.WriteString(s)
	} else {
		w.WriteByte(b)
	}
}

func highlightSyntax(text string) []byte {
	var w codeWriter
	w.WriteString(`<code class="code-tag">`)

	buf := []byte(text)
	token := make([]byte, 0, 64)
	typ := unmatched
	var prev, next byte

	for i := 0; i < len(buf); i++ {
		b := buf[i]
		if i != len(buf)-1 {
			next = buf[i+1]
		} else {
			next = 0
		}

		switch typ {
		case unmatched:
			switch b {
			case '/':
				if next == '/' {
					typ = comment
					w.WriteString(commentHeader)
					w.WriteString(`//`)
					i++
				} else {
					w.wrapOperator(b)
				}
			case '#':
				typ = comment
				w.WriteString(commentHeader)
				w.WriteByte(b)
			case '\'':
				typ = quoted
				w.WriteString(stringHeader)
				w.WriteString("&#39;")
			case '"':
				typ = doubleQuoted
				w.WriteString(stringHeader)
				w.WriteString("&#34;")
			default:
				switch {
				case operators[b]:
					w.wrapOperator(b)
				case isWordByte(b):
					typ = word
					token = append(token, b)
				default:
					w.escapeByte(b)
				}
			}
		case word:
			token = append(token, b)
			if !isWordByte(next) {
				switch {
				case next == '(':
					w.WriteString(functionHeader)
					w.escape(token)
					w.close()
				case keywords[string(token)]:
					w.WriteString(operatorHeader)
					w.Write(token)
					w.close()
				default:
					w.escape(token)
				}
				typ = unmatched
				token = token[0:0]
			}
		case quoted:
			w.escapeByte(b)
			if b == '\'' && prev != '\\' {
				w.close()
				typ = unmatched
			}
		case doubleQuoted:
			w.escapeByte(b)
			if b == '"' && prev != '\\' {
				w.close()
				typ = unmatched
			}
		case comment:
			w.escapeByte(b)
		}

		prev = b
	}

	if typ != unmatched {
		w.close()
	}
	w.WriteString("</code>")
	return w.Bytes()
}

// close open tag
func (w *codeWriter) close() {
	w.WriteString(`</span>`)
}

func (w *codeWriter) wrapOperator(b byte) {
	w.WriteString(operatorHeader)
	w.escapeByte(b)
	w.close()
}

// Returns if byte is an ASCII alphanumeric, $ or _
func isWordByte(b byte) bool {
	return b == 36 ||
		(b >= 48 && b <= 57) ||
		(b >= 65 && b <= 90) ||
		b == 95 ||
		(b >= 97 && b <= 122)
}
