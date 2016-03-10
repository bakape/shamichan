interface Document {
	query(selector: string): Element
	queryAll(selector: string): Elements
}

interface Element {
	closest(selector: string): Element
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
