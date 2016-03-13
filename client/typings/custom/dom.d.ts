interface Document {
	query(selector: string): Element
	queryAll(selector: string): Elements
}

interface Element {
	disabled?: boolean

	append(...nodes: (Node|string)[]): void
	closest(selector: string): Element
	matches(selector: string): boolean
	query(selector: string): Element
	queryAll(selector: string): Elements
}

declare class Elements extends Array<Element> {
	query(selector: string): Element
	queryAll(selector: string): Elements
}

interface EventTarget {
	matches?(selector: string): boolean
}
