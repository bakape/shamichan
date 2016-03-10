/*
 Utuility functions.
*/

// Fetches a JSON response from the API and returns a Promise
export function fetchJSON(url: string): Promise<Object> {
	return fetch("api/" + url).then(res => res.json())
}

// Generate a random alphannumeric string of lower and upper case hexadecimal
// characters
export function randomID(len: number): string {
	let id = ''
	for (let i = 0; i < len; i++) {
		let char = (Math.random() * 36).toString(36)[0]
		if (Math.random() < 0.5) {
			char = char.toUpperCase()
		}
		id += char
	}
	return id
}

// Simple map of sets with automatic array creation and removal
export class SetMap<V> {
	private map: {[key: string]: Set<V>} = {}

	// Add item to key
	add(key: string, item: V) {
		if (!(key in this.map)) {
			this.map[key] = new Set()
		}
		this.map[key].add(item)
	}

	// Remove and item from a key
	remove(key: string, item: V) {
		const set = this.map[key]
		if (!set) {
			return
		}
		set.delete(item)
		if (set.size === 0) {
			delete this.map[key]
		}
	}

	// Execute a function for each item under a key
	forEach(key: string, fn: (item: V) => void) {
		const set = this.map[key]
		if (!set) {
			return
		}
		set.forEach(fn)
	}
}

// Retrieve post number of post element
export function getNum(el: Element): number {
	if (!el) {
		return 0
	}
	return parseInt(el.getAttribute('id').slice(1), 10)
}

// Retrieve post number of closest parent post element
export function getID(el: Element): number {
	if (!el) {
		return 0
	}
	return getNum(el.closest('article, section'))
}

// Parse HTML string to node array
export function parseEls(DOMString: string): Node[] {
	const el = document.createElement('div')
	el.innerHTML = DOMString
	return Array.from(el.childNodes)
}

// Parse HTML string to a single Node
export function parseEl(DOMString: string): Node {
	const el = document.createElement('div')
	el.innerHTML = DOMString
	return el.firstChild
}

// Add an event listener that filters targets according to a CSS selector
export function onEvent(
	el: Element,
	type: string,
	selector: string,
	handler: EventListener
) {
	el.addEventListener(type, event => {
		if (event.target.matches(selector)) {
			handler(event)
		}
	})
}

// Add event listener to element, that will only be executed once
export function once(el: Element, type: string, handler: EventListener) {
	el.addEventListener(type, event => {
		handler(event)
		el.removeEventListener(type, handler)
	})
}

// Return width of element with padding and margin
export function outerWidth(el: Element): number {
	const style =  getComputedStyle(el),
		props = ['marginLeft', 'marginRight', 'paddingLeft','paddingRight']
	let width = 0
	for (let prop of props) {
		width += parseInt(style[prop]);
	}
	return width
}

// Confirms email is saging
export function isSage(email: string) :boolean {
	if (email) {
		return email.trim() === 'sage'
	}
	return false
}

// Pad an integer with a leading zero, if below 10
export function pad(n: number): string {
	return (n < 10 ? '0' : '') + n
}

// Template string tag function for HTML. Strips indentation and trailing
// newlines. Based on https://gist.github.com/zenparsing/5dffde82d9acef19e43c
export function parseHTML(callSite: string[], ...args: string[]): string {
	let output = callSite[0]
	for (let i = 1; i <= args.length; i++) {
		output += args[i - 1] + callSite[i]
	}

	// Strip indentation and remove empty lines from HTML string
	return output.replace(/\s*\n\s*/g, '')
}

type ElementAttributes = {[key: string]: string}

// Generate an HTML element attribute list. If a key has an empty string, it's
// value will be considered "true"
export function parseAttributes(attrs: ElementAttributes): string {
	let html = ''
	for (let key in attrs) {
		html += ' '
		const val = attrs[key]
		if (val) {
			html += `${key}="${val}"`
		} else {
			html += key
		}
	}
	return html
}

// Makes a ', ' seperated list
export function commaList(items: string[]): string {
	let html = ''
	for (let item of items) {
		if (html) {
			html += ', '
		}
		html += item
	}
	return html
}
