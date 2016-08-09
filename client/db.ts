/*
 IndexedDB database controller
*/

const dbVersion = 3

export let db: IDBDatabase

// Execute a database request as a promise
IDBRequest.prototype.exec = function (): Promise<any> {
	return new Promise<any>((resolve, reject) => {
		this.onerror = () =>
			reject(this.error)
		this.onsuccess = () =>
			resolve(this.result)
	})
}

// Open a connection to the IndexedDB database
export function open(): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		const r = indexedDB.open('meguca', dbVersion)
		r.onerror = () =>
			reject(r.error)
		r.onsuccess = () => {
			db = r.result
			db.onerror = err => {
				throw err
			}
			resolve()
		}
		r.onupgradeneeded = event => {
			const db = r.result as IDBDatabase

			// Various post number sets, like posts the user has made, posts
			// that have qouted the user, posts that have been hidded, etc.
			const posts = db.createObjectStore('posts', {keyPath: 'id'})

			posts.add({id: 'mine'}) // Posts this client has made
			posts.add({id: 'hidden'}) // Posts this client has hidden

			// Variuos miisceleneous objects
			const main = db.createObjectStore('main', {keyPath: 'id'})
			main.add({id: 'background'})
		}
	})
}
