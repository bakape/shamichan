interface IDBRequest {
	exec(): Promise<any>
}

interface ChildNode {
	after(...nodes: (Node|string)[]): void
	before(...nodes: (Node | string)[]): void
	replaceWith(...nodes: (Node|string)[]): void
}

interface ParentNode {
	append(...nodes: (Node|string)[]): void
	prepend(...nodes: (Node|string)[]): void
}

interface NodeSelector {
	querySelector(sel: string): Element

	// Hack. Modern browsers have Symbol.iterator on NodeList
	querySelectorAll(sel: string): Element[]
}

interface EventTarget {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
}

interface EventListenerOptions {
	capture?: boolean
	once?: boolean
	passive?: boolean
}

interface Element extends ChildNode, ParentNode {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
}

interface HTMLInputElement {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
}

interface Node extends ChildNode, ParentNode {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean|EventListenerOptions
	): void
}