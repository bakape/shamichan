/*
 IndexedDB manager
*/

const dbVersion = 2

let db :IDBDatabase

declare class IDBRequest {
	exec(): Promise<any>
}

// Execute a database request as a promise
IDBRequest.prototype.exec = function () {
	return new Promise((resolve, reject) => {
		this.onerror = () => reject(this.error)
		this.onsuccess = () => resolve(this.result)
	})
}

export function open(): Promise<{}> {
	return new Promise((resolve, reject) => {
		const r = indexedDB.open('meguca', dbVersion)
		r.onerror = () =>
			reject(r.error)
		r.onsuccess = () =>
			db = r.result
		r.onupgradeneeded = event => {
			const db = r.result

			// Stores user-set settings
			db.createObjectStore('options', {keyPath: 'id'})

			// Various post number sets, like posts the user has made, posts
			// that have qouted the user, posts that have been hidded, etc.
			db.createObjectStore('posts', {keyPath: 'id'})

			// Chache of thread models, so we don't have to store JSON and
			// reparse it, when restoring to a previous state
			db.createObjectStore('threads', {keyPath: 'id'})

			// Same for boards
			db.createObjectStore('boards', {keyPath: 'id'})
		}
	})
}
