/*
 Utuility functions used both by the client and worker
*/

// Message types for browser tab <-> serviceWorker communication
export const enum message {
	disconnect,
	syncStatus
}

// Tokenization map of sync statuses
export const enum syncStatus  {disconnected, syncing, synced}

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
export class SetMap<K, V> {
	private map: Map<K, Set<V>>

	constructor() {
		this.map = new Map()
	}

	// Add item to key
	add(key: K, item: V) {
		if (!this.map.has(key)) {
			this.map.set(key, new Set())
		}
		this.map.get(key).add(item)
	}

	// Remove and item from a key
	remove(key: K, item: V) {
		const set = this.map.get(key)
		if (!set) {
			return
		}
		set.delete(item)
		if (set.size === 0) {
			this.map.delete(key)
		}
	}

	// Execute a function for each item under a key
	forEach(key: K, fn: (item: V) => void) {
		const set = this.map.get(key)
		if (!set) {
			return
		}
		set.forEach(fn)
	}
}
