interface Node {
	children: HTMLCollection

	after(...nodes: (Node|string)[]): void
	append(...nodes: (Node|string)[]): void
	before(...nodes: (Node|string)[]): void
	closest(selector: string): Element
	prepend(...nodes: (Node|string)[]): void
	remove(): void
	replaceWith(...nodes: (Node|string)[]): void
}

interface Element {
	hidden: boolean
	style: CSSStyleDeclaration

	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
	click(): void
	focus(): void
	matches(selector: string): boolean
}

interface HTMLInputElement {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
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
