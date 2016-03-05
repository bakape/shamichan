/*
 Utuility functions.
*/

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
export class SetMap<V> {
	private map: {[key: string]: Set<V>} = {}

	// Add item to key
	add(key: string, item: V) {
		if (!(key in this.map)) {
			this.map[key] = new Set()
		}
		this.map[key].add(item)
	}

	// Remove and item from a key
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
