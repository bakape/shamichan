import { escape } from "../../util"

const enum type { unmatched, word, quoted, doubleQuoted, comment }

const headers = {
	operator: "<span class=\"ms-operator\">",
	function: "<span class=\"ms-function\">",
	string: "<span class=\"ms-string\">",
	comment: "<span class=\"ms-comment\">",
}

const close = "</span>"

const keywords = {
	"NULL": true,
	"NaN": true,
	"abstract": true,
	"alias": true,
	"and": true,
	"arguments": true,
	"array": true,
	"asm": true,
	"assert": true,
	"async": true,
	"auto": true,
	"await": true,
	"base": true,
	"begin": true,
	"bool": true,
	"boolean": true,
	"break": true,
	"byte": true,
	"case": true,
	"catch": true,
	"char": true,
	"checked": true,
	"class": true,
	"clone": true,
	"compl": true,
	"const": true,
	"continue": true,
	"debugger": true,
	"decimal": true,
	"declare": true,
	"default": true,
	"defer": true,
	"deinit": true,
	"delegate": true,
	"delete": true,
	"do": true,
	"double": true,
	"echo": true,
	"elif": true,
	"else": true,
	"elseif": true,
	"elsif": true,
	"end": true,
	"ensure": true,
	"enum": true,
	"event": true,
	"except": true,
	"exec": true,
	"explicit": true,
	"export": true,
	"extends": true,
	"extension": true,
	"extern": true,
	"fallthrough": true,
	"false": true,
	"final": true,
	"finally": true,
	"fixed": true,
	"float": true,
	"fn": true,
	"for": true,
	"foreach": true,
	"friend": true,
	"from": true,
	"func": true,
	"function": true,
	"global": true,
	"go": true,
	"goto": true,
	"guard": true,
	"if": true,
	"impl": true,
	"implements": true,
	"implicit": true,
	"import": true,
	"in": true,
	"int": true,
	"include": true,
	"inline": true,
	"inout": true,
	"instanceof": true,
	"interface": true,
	"internal": true,
	"is": true,
	"lambda": true,
	"let": true,
	"lock": true,
	"long": true,
	"module": true,
	"mut": true,
	"mutable": true,
	"namespace": true,
	"native": true,
	"new": true,
	"next": true,
	"nil": true,
	"not": true,
	"null": true,
	"object": true,
	"operator": true,
	"or": true,
	"out": true,
	"override": true,
	"package": true,
	"params": true,
	"private": true,
	"protected": true,
	"protocol": true,
	"pub": true,
	"public": true,
	"raise": true,
	"readonly": true,
	"redo": true,
	"ref": true,
	"register": true,
	"repeat": true,
	"require": true,
	"rescue": true,
	"restrict": true,
	"retry": true,
	"return": true,
	"sbyte": true,
	"sealed": true,
	"short": true,
	"signed": true,
	"sizeof": true,
	"static": true,
	"str": true,
	"string": true,
	"struct": true,
	"subscript": true,
	"super": true,
	"switch": true,
	"synchronized": true,
	"template": true,
	"then": true,
	"throws": true,
	"transient": true,
	"true": true,
	"try": true,
	"type": true,
	"typealias": true,
	"typedef": true,
	"typeid": true,
	"typename": true,
	"typeof": true,
	"uint": true,
	"unchecked": true,
	"undef": true,
	"undefined": true,
	"union": true,
	"unless": true,
	"unsigned": true,
	"until": true,
	"use": true,
	"using": true,
	"var": true,
	"virtual": true,
	"void": true,
	"volatile": true,
	"when": true,
	"where": true,
	"while": true,
	"with": true,
	"xor": true,
	"yield": true,
}

const operators = {
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

export default function highlightSyntax(text: string): string {
	let html = `<code class="code-tag">`,
		token = "",
		typ = type.unmatched

	for (let i = 0; i < text.length; i++) {
		const prev = text[i - 1] || "",
			b = text[i],
			next = text[i + 1] || ""

		switch (typ) {
			case type.unmatched:
				switch (b) {
					case "/":
						if (next === "/") {
							typ = type.comment
							html += headers.comment + "//"
							i++
						} else {
							html += wrapOperator(b)
						}
						break
					case "#":
						typ = type.comment
						html += headers.comment + b
						break
					case "'":
						typ = type.quoted
						html += headers.string + "&#39;"
						break
					case "\"":
						typ = type.doubleQuoted
						html += headers.string + "&#34;"
						break
					default:
						if (operators[b]) {
							html += wrapOperator(b)
						} else if (isWordByte(b)) {
							typ = type.word
							token += b
						} else {
							html += escape(b)
						}
				}
				break
			case type.word:
				token += b
				if (!isWordByte(next)) {
					if (next === "(") {
						html += headers.function + escape(token) + close
					} else if (keywords[token]) {
						html += headers.operator + token + close
					} else {
						html += escape(token)
					}
					typ = type.unmatched
					token = ""
				}
				break
			case type.quoted:
				html += escape(b)
				if (b === "'" && prev != "\\") {
					html += close
					typ = type.unmatched
				}
				break
			case type.doubleQuoted:
				html += escape(b)
				if (b === "\"" && prev != "\\") {
					html += close
					typ = type.unmatched
				}
				break
			case type.comment:
				html += escape(b)
				break
		}
	}
	if (typ !== type.unmatched) {
		html += close
	}
	html += "</code>"
	return html
}

function wrapOperator(b: string): string {
	return headers.operator + escape(b) + close
}

// Returns if byte is an ASCII alphanumeric, $ or _
function isWordByte(ch: string): boolean {
	const b = ch.charCodeAt(0)
	return b == 36 ||
		(b >= 48 && b <= 57) ||
		(b >= 65 && b <= 90) ||
		b == 95 ||
		(b >= 97 && b <= 122)
}
