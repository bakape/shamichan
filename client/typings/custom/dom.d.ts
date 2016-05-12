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

	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
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
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
	matches?(selector: string): boolean
}

type EventListenerOptions = {
	capture?: boolean
	once?: boolean
	passive?: boolean
}
