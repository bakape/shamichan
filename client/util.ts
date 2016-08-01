import {write, read} from './render'
import {BoardConfigs} from './state'

type AnyHash = {[key: string]: any}

// Single entry of the array, fetched through `/json/boardList`
export type BoardEntry = {
	id: string
	title: string
}

const base64 =
	'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_'
	.split("")

// Fetches and decodes a JSON response from the API
export const fetchJSON = async (url: string): Promise<any> =>
	await (await fetch(url)).json()

// Returns a list of all boards created in alphabetical order
export const fetchBoardList = async (): Promise<BoardEntry[]> =>
	((await fetchJSON("/json/boardList") as BoardEntry[]))
	.sort((a, b) =>
		a.id.localeCompare(b.id))

// Fetch configurations of a specific board
export const fetchBoarConfigs = async (board: string): Promise<BoardConfigs> =>
	await fetchJSON(`/json/boardConfig/${board}`)

// Generate a random base64 string of desird length
export function randomID(len: number): string {
	let id = ''
	for (let i = 0; i < len; i++) {
		id += random(base64)
	}
	return id
}

// Return a random item from an array
export function random<T>(array: T[]): T {
	return array[Math.floor(Math.random() * array.length)]
}

// Simple map of sets with automatic array creation and removal
export class SetMap<V> {
	map: {[key: string]: Set<V>} = {}

	// Add item to key
	add(key: string, item: V) {
		if (!(key in this.map)) {
			this.map[key] = new Set()
		}
		this.map[key].add(item)
	}

	// Remove an item from a key
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
export const getNum = (el: Element): number =>
	el ? parseInt(el.getAttribute('id').slice(1), 10) : 0

// Retrieve post number of closest parent post element
export const getID = (el: Element): number =>
	el ? getNum(el.closest('article, section')) : 0

// Parse HTML string to node array
export function makeEls(DOMString: string): Node[] {
	const el = document.createElement('div')
	el.innerHTML = DOMString
	return Array.from(el.childNodes)
}

// Parse HTML string to a single Node
export function makeEl(DOMString: string): Node {
	const el = document.createElement('div')
	el.innerHTML = DOMString
	return el.firstChild
}

export interface OnOptions extends EventListenerOptions {
	selector?: string
}

// Add an event listener that optionally filters targets according to a CSS
// selector.
export function on(
	el: Element,
	type: string,
	fn: EventListener,
	opts?: OnOptions
) {
	if (opts && opts.selector) {
		const oldFn = fn
		fn = event => {
			if (event.target.matches(opts.selector)) {
				oldFn(event)
			}
		}
	}
	el.addEventListener(type, fn, opts)
}

// Return width of element with padding and margin
export function outerWidth(el: Element): number {
	const style =  getComputedStyle(el)
	const widths = [
		style.marginLeft, style.marginRight, style.paddingLeft,
		style.paddingRight
	]
	let total = 0
	for (let width of widths) {
		total += parseInt(width)
	}
	return total
}

// Pad an integer with a leading zero, if below 10
export const pad = (n: number): string =>
	(n < 10 ? '0' : '') + n

// Template string tag function for HTML. Strips indentation and trailing
// newlines. Based on https://gist.github.com/zenparsing/5dffde82d9acef19e43c
export function HTML(callSite: string[], ...args: string[]): string {
	let output = callSite[0]
	for (let i = 1; i <= args.length; i++) {
		output += args[i - 1] + callSite[i]
	}

	// Strip indentation and remove empty lines from HTML string
	return output.replace(/\s*\n\s*/g, '')
}

// Generate an HTML element attribute list. If a key has an empty string, it's
// value will be considered "true"
export function makeAttrs(attrs: StringMap): string {
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

// Copy all properties from the source object to the destination object
export function extend(dest: {}, source: {}) {
	for (let key in source) {
		(dest as AnyHash)[key] = (source as AnyHash)[key]
	}
}

// Remove values from the array, that do not pass the truth test
export function filter<T>(array: T[], fn: (item: T) => boolean): T[] {
	const filtered: T[] = []
	for (let item of array) {
		if (fn(item)) {
			filtered.push(item)
		}
	}
	return filtered
}

// Group all objects in array by a property of the object
export function groupBy<T extends Object>(array: T[], prop: string)
	: {[key: string]: T[]}
{
	const groups: {[key: string]: T[]} = {}
	for (let item of array) {
		const dest = (item as AnyHash)[prop]
		if (dest in groups) {
			groups[dest].push(item)
		} else {
			groups[dest] = [item]
		}
	}
	return groups
}

// Returns the first element of an array, that passes the truth test, or
// undefined
export function find<T>(arrayLike: ArrayLike<T>, fn: (item: T) => boolean): T {
	for (let i = 0; i < arrayLike.length; i++) {
		if (fn(arrayLike[i])) {
			return arrayLike[i]
		}
	}
	return undefined
}

// Iterates over an array-like object, like HTMLCollection
export function each<T>(arrayLike: ArrayLike<T>, fn: (item: T) => void) {
	for (let i = 0; i < arrayLike.length; i++) {
		fn(arrayLike[i])
	}
}

interface Loader {
	onload: EventListener
	onerror: EventListener
}

// Wraps event style object with onload() method to Promise style
export const load = (loader: Loader): Promise<Event> =>
	new Promise<Event>((resolve, reject) => {
		loader.onload = resolve
		loader.onerror = reject
	})

// Dynamically lead a System module
export function loadModule(path: string): Promise<any> {
	path = `es${(window as any).legacy ? 5 : 6}/${path}`
	return System.import(path)
}

const escapeMap: {[key: string]: string} = {
	'&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#x27;',
    '`': '&#x60;',
}

// Escape a user-submitted unsafe string to protect against XSS.
export const escape = (str: string): string =>
	str.replace(/[&<>'"`]/g , char =>
		escapeMap[char])

// Construct a table from an array of objects and a consumer funtion,
// that returns an array of cells.
export function table<T>(rows: T[], func: (arg: T) => string[]): string {
	let html = '<table>'
	for (let row of rows) {
		html += '<tr>'
		for (let cell of func(row)) {
			html += `<td>${cell}</td>`
		}
		html += '</tr>'
	}
	html += '</table>'
	return html
}

// Extract the value of a named input field, which is a child of the parameter
// element
export const inputValue = (el: Element, name: string): string =>
	(el.querySelector(`input[name=${name}]`) as HTMLInputElement).value

// Applies mixins to destination object's prototype
export const applyMixins = (dest: any, ...mixins: any[]) =>
	mixins.forEach(mixin =>
		Object.getOwnPropertyNames(mixin.prototype).forEach(name =>
			dest.prototype[name] = mixin.prototype[name]))
