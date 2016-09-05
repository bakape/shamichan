// IndexedDB database controller

const dbVersion = 4

export let db: IDBDatabase

// Expiring post ID object stores
const postStores = [
	"mine",   // Posts created by this client
	"hidden", // Posts hidden by client
	"seen",   // Contains last post seen in a thread
]

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

		// Upgrade the database
		r.onupgradeneeded = event => {
			const db = (event.target as any).result as IDBDatabase

			// Delete all previous object stores
			for (let name of Array.from(db.objectStoreNames)) {
				db.deleteObjectStore(name)
			}

			for (let name of postStores) {
				db
					.createObjectStore(name, {autoIncrement: true})
					.createIndex("expires", "expires")
			}

			// Variuos miisceleneous objects
			const main = db.createObjectStore('main', {keyPath: 'id'})
			main.add({id: 'background'})
		}

		// Prepare for operation
		r.onsuccess = () => {
			db = r.result as IDBDatabase

			db.onerror = throwErr
			resolve()
			for (let name of postStores) {
				deleteExpired(name)
			}

			// Reload this tab, if another tab requires a DB upgrade
			db.onversionchange = () =>
				(db.close(),
				location.reload(true))
		}
	})
}

function throwErr(err: ErrorEvent) {
	throw err
}

// Delete expired keys from post ID object stores
function deleteExpired(name: string) {
	const trans = db.transaction(name, "readwrite")
	trans.onerror = throwErr

	const range = IDBKeyRange.upperBound(Date.now()),
		req = trans.objectStore(name).index("expires").openCursor(range)
	req.onerror = throwErr

	req.onsuccess = event => {
		const cursor = (event.target as any).result as IDBCursor
		if (!cursor) {
			return
		}
		cursor.delete()
		cursor.continue()
	}
}

// Asynchronously insert a new expiring object into a postStore
export function storeID(objStore: string, expiry: number, obj: {}) {
	const trans = db.transaction(objStore, "readwrite")
	trans.onerror = throwErr

	; (obj as any).expires = Date.now() + expiry
	const req = trans.objectStore(objStore).add(obj)
	req.onerror = throwErr
}

