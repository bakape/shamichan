interface Node {
	children: HTMLCollection
}

interface Element {
	disabled?: boolean
	checked?: boolean
	value?: any
	files?: FileList
	children: HTMLCollection
	style: CSSStyleDeclaration

	after(...nodes: (Node|string)[]): void
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
	append(...nodes: (Node|string)[]): void
	before(...nodes: (Node|string)[]): void
	click(): void
	closest(selector: string): Element
	matches(selector: string): boolean
	prepend(...nodes: (Node|string)[]): void
	replaceWith(...nodes: (Node|string)[]): void
}

interface EventTarget {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
	matches?(selector: string): boolean
}

interface EventListenerOptions {
	capture?: boolean
	once?: boolean
	passive?: boolean
}
