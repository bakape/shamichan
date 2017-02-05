// Various utility functions

export { default as FSM } from "./fsm"
export * from "./fetch"
export * from "./hooks"
export * from "./scroll"
export * from "./render"
export * from "./changes"

// Options for the on() addEventListener() wrapper
export interface OnOptions extends EventListenerOptions {
	selector?: string
}

// Any object with an event-based interface for passing to load()
interface Loader {
	onload: EventListener
	onerror: EventListener
}

// Retrieve post id of post element
function getID(el: Element): number {
	if (!el) {
		return 0
	}
	return parseInt(el.getAttribute('id').slice(1), 10)
}

// Retrieve post number of closest parent post element
export function getClosestID(el: Element): number {
	if (!el) {
		return 0
	}
	return getID(el.closest('article'))
}

// Parse HTML string to node array
export function makeFrag(DOMString: string): DocumentFragment {
	const el = document.createElement("template") as HTMLTemplateElement
	el.innerHTML = DOMString
	return el.content
}

// Add an event listener that optionally filters targets according to a CSS
// selector.
export function on(
	el: EventTarget,
	type: string,
	fn: EventListener,
	opts?: OnOptions
) {
	if (opts && opts.selector) {
		const oldFn = fn
		fn = event => {
			if ((event.target as Element).matches(opts.selector)) {
				oldFn(event)
			}
		}
	}
	el.addEventListener(type, fn, opts)
}

// Pad an integer with a leading zero, if below 10
export function pad(n: number): string {
	return (n < 10 ? '0' : '') + n
}

// Template string tag function for HTML. Strips indentation and newlines.
export function HTML(base: TemplateStringsArray, ...args: string[]): string {
	let output = base[0]
	for (let i = 1; i <= args.length; i++) {
		output += args[i - 1] + base[i]
	}
	return output.replace(/[\t\n]+/g, '')
}

// Generate an HTML element attribute list. If a key has an empty string, it's
// value will be considered "true"
export function makeAttrs(attrs: { [key: string]: string }): string {
	let html = ''
	for (let key in attrs) {
		html += ' ' + key
		const val = attrs[key]
		if (val) {
			html += `="${val}"`
		}
	}
	return html
}

// Set attributes from a key-value map to the element
export function setAttrs(el: Element, attrs: { [key: string]: string }) {
	for (let key in attrs) {
		el.setAttribute(key, attrs[key])
	}
}

// Copy all properties from the source object to the destination object. Nested
// objects are extended recursively.
export function extend(dest: {}, source: {}) {
	for (let key in source) {
		const val = source[key]
		if (typeof val === "object" && val !== null) {
			const d = dest[key]
			if (d) {
				extend(d, val)
			} else {
				dest[key] = val
			}
		} else {
			dest[key] = val
		}
	}
}

// Wraps event style object with onload() method to Promise style
export function load(loader: Loader): Promise<Event> {
	return new Promise<Event>((resolve, reject) => {
		loader.onload = resolve
		loader.onerror = reject
	})
}

const escapeMap: { [key: string]: string } = {
	'&': '&amp;',
	'<': '&lt;',
	'>': '&gt;',
	'"': '&quot;',
	"'": '&#x27;',
	'`': '&#x60;',
}

// Escape a user-submitted unsafe string to protect against XSS.
export function escape(str: string): string {
	return str.replace(/[&<>'"`]/g, char =>
		escapeMap[char])
}

// Extract the value of a named input field, which is a child of the parameter
// element
export function inputValue(el: Element, name: string): string {
	return (el.querySelector(`input[name=${name}]`) as HTMLInputElement)
		.value
}

// Return either the singular or plural form of a translation, depending on
// number
export function pluralize(num: number, word: [string, string]): string {
	return `${num} ${word[num === 1 || num === -1 ? 0 : 1]}`
}

// Return width of element with padding and margin
export function outerWidth(el: HTMLElement): number {
	const style = getComputedStyle(el)
	const widths = [
		style.marginLeft, style.marginRight, style.paddingLeft,
		style.paddingRight
	]
	let total = el.offsetWidth
	for (let width of widths) {
		total += parseInt(width)
	}
	return total
}
