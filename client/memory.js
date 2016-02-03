import {isEmpty, size} from 'underscore'

// All instances of the Memory class
const memories = {}

// Listen for storage events and update the stored value for exising memory
// instances, if the key changes. These only fire, if the write happens in
// another tab of the same origin.
window.addEventListener('storage', ({key, newValue}) => {
	if (key in memories) {
		memories[key].cached = parseSet(newValue)
	}
})

// Parse a stringified set
function parseSet(set) {
	let val = {}
	try {
		val = JSON.parse(set)
	}
	catch(e) {}
	return val
}

// Self-expiring localStorage set manager
export default class Memory {
	constructor(key, expiry) {
		this.key = key
		memories[key] = this
		this.expiry = expiry

		// Read the initial value
		this.cached = this.read()

		// Purge old entries on start
		setTimeout(() => this.purgeExpired(), 5000)
	}

	// Return current time in seconds
	now() {
		return Math.floor(Date.now() / 1000)
	}

	// Clear the stored set
	purgeAll() {
		localStorage.removeItem(this.key)
	}

	// Read and parse the stringified set from localStorage
	read() {
		const key = localStorage.getItem(this.key)
		if (!key) {
			return {}
		}
		return parseSet(key)
	}

	// Return, if the given key exists in the set
	has(key) {
		return !!this.cached[key]
	}

	// Replace the existing set, if any, with the suplied one
	writeAll(set) {
		if (isEmpty(set)) {
			return this.purgeAll()
		}
		localStorage.setItem(this.key, JSON.stringify(set))
	}

	// Write a single key to the stored set
	write(key) {
		// When performing writes, best fetch everything, rather than rely on
		// events for browser tab cache synchronisation. Browser backround tab
		// optimisation might fuck us over.
		this.cached = this.read()
		this.cached[key] = this.now()
		this.writeAll(this.cached)
		return size(this.cached) // Return number of items
	}

	// Return the current size of the stored Set
	size() {
		return size(this.cached)
	}

	// Delete expired entries from set and write to localStorage
	purgeExpired() {
		this.cached = this.read()
		const now = this.now(),
			limit = 86400 * this.expiry,
			expired = []
		for (let key in this.cached) {
			if (now > this.cached[key] + limit) {
				expired.push(key)
			}
		}
		if (!expired.length) {
			return
		}
		for (let key of expired) {
			delete this.cached[key]
		}
		this.writeAll(this.cached)
	}
}
