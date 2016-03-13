interface Document {
	query(selector: string): Element
	queryAll(selector: string): Elements
}

interface Node {
	children: HTMLCollection
}

interface Element {
	disabled?: boolean
	checked?: boolean
	value?: any
	children: HTMLCollection
	files?: FileList
	style: CSSStyleDeclaration

	append(...nodes: (Node|string)[]): void
	click(): void
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
