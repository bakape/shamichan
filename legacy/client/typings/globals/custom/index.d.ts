interface EventTarget {
	addEventListener(
		type: string,
		handler: EventListener,
		options?: boolean | EventListenerOptions
	): void
}

interface EventListenerOptions {
	capture?: boolean
	once?: boolean
	passive?: boolean
}

interface ArrayBufferTarget extends EventTarget {
	result: ArrayBuffer
}

interface ArrayBufferLoadEvent extends Event {
	target: ArrayBufferTarget
}

interface NotificationOptions {
	sticky?: boolean;
}
