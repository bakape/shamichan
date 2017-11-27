// IndexedDB database controller

const dbVersion = 7

let db: IDBDatabase

// Database has errored and all future calls should be ignored
// FF IndexedDB implementation is broken in private mode.
// See https://bugzilla.mozilla.org/show_bug.cgi?id=781982
// This helps bypass this.
let hasErrored = false;

// Expiring post ID object stores
const postStores = [
	"mine",     // Posts created by this client
	"hidden",   // Posts hidden by client
	"seen",     // Replies to the user's posts that have already been seen
	"seenPost", // Posts that the user has viewed or scrolled past
]

// Open a connection to the IndexedDB database
export function open(): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		const r = indexedDB.open('meguca', dbVersion)

		r.onerror = () =>
			reject(r.error)

		r.onupgradeneeded = upgradeDB

		// Prepare for operation
		r.onsuccess = () => {
			db = r.result as IDBDatabase

			db.onerror = throwErr
			resolve()

			// Reload this tab, if another tab requires a DB upgrade
			db.onversionchange = () =>
				(db.close(),
					location.reload(true))

			// Delay for quicker starts
			setTimeout(() => {
				for (let name of postStores) {
					deleteExpired(name)
				}
			}, 10000)
		}
	})
		.catch(err => {
			hasErrored = true
			console.error("Error loading IndexedDB. All further DB access will be ignored")
			console.error(err)
		})
}

// Upgrade or initialize the database
function upgradeDB(event: IDBVersionChangeEvent) {
	db = (event.target as any).result as IDBDatabase
	switch (event.oldVersion) {
		case 0:
		case 1:
		case 2:
		case 3:
			// Delete all previous object stores
			for (let name of Array.from(db.objectStoreNames)) {
				db.deleteObjectStore(name)
			}

			for (let name of postStores) {
				const s = db.createObjectStore(name, { autoIncrement: true })
				s.createIndex("expires", "expires")
				s.createIndex("op", "op")
			}

			// Various miscellaneous objects
			const main = db.createObjectStore('main', { keyPath: 'id' })
			main.add({ id: 'background' })
			main.add({ id: 'mascot' })
			break
		case 4:
			// Can't modify data during an upgrade, so do it right after the
			// "upgrade" completes
			setTimeout(() => addObj("main", { id: "mascot" }), 1000)
			break
		case 5:
			if (db.objectStoreNames.contains("seenPost")) {
				break
			}
			db.createObjectStore("seenPost", { autoIncrement: true })
				.createIndex("expires", "expires")
			break
		case 6:
			// Recreate all previous post ID stores
			for (let name of postStores) {
				db.deleteObjectStore(name)
				const s = db.createObjectStore(name, { autoIncrement: true })
				s.createIndex("expires", "expires")
				s.createIndex("op", "op")
			}
			break
	}
}

// Helper for throwing errors with event-based error passing
function throwErr(err: ErrorEvent) {
	throw err
}

// Delete expired keys from post ID object stores
function deleteExpired(name: string) {
	const req = newTransaction(name, true)
		.index("expires")
		.openCursor(IDBKeyRange.upperBound(Date.now()))

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

// Helper for initiating transactions on a single object store
function newTransaction(store: string, write: boolean): IDBObjectStore {
	const t = db.transaction(store, write ? "readwrite" : "readonly")
	t.onerror = throwErr
	return t.objectStore(store)
}

// Read the contents of a postStore for specific threads into an array
export function readIDs(store: string, ops: number[]): Promise<number[]> {
	if (hasErrored || !ops.length) {
		return fakePromise([])
	}
	return Promise.all(
		ops.map(id =>
			readThreadIDs(store, id))
	)
		.then(ids =>
			[].concat(...ids))
}

// Reads post IDs for a single thread
function readThreadIDs(store: string, op: number): Promise<number[]> {
	return new Promise<number[]>((resolve, reject) => {
		const ids: number[] = [],
			req = newTransaction(store, false)
				.index("op")
				.openCursor(IDBKeyRange.bound(op, op))

		req.onerror = err =>
			reject(err)

		req.onsuccess = event => {
			const cursor = (event as any).target.result as IDBCursorWithValue
			if (cursor) {
				ids.push(cursor.value.id)
				cursor.continue()
			} else {
				resolve(ids)
			}
		}
	})
}

function fakePromise<T>(res: T): Promise<T> {
	return new Promise(r =>
		r(res))
}

// Asynchronously insert a new expiring post id object into a postStore
export function storeID(store: string, id: number, op: number, expiry: number) {
	if (hasErrored) {
		return
	}
	addObj(store, {
		id, op,
		expires: Date.now() + expiry,
	})
}

function addObj(store: string, obj: any) {
	newTransaction(store, true).add(obj).onerror = throwErr
}

// Clear the target object store asynchronously
export function clearStore(store: string) {
	if (hasErrored) {
		return
	}
	const trans = newTransaction(store, true),
		req = trans.clear()
	req.onerror = throwErr
}

// Retrieve an object from a specific object store
export function getObj<T>(store: string, id: any): Promise<T> {
	if (hasErrored) {
		return fakePromise({} as any)
	}
	return new Promise<T>((resolve, reject) => {
		const t = newTransaction(store, false),
			r = t.get(id)
		r.onerror = () =>
			reject(r.error)
		r.onsuccess = () =>
			resolve(r.result)
	})
}

// Put an object in the specific object store
export function putObj(store: string, obj: any): Promise<void> {
	if (hasErrored) {
		return fakePromise(undefined)
	}
	return new Promise<void>((resolve, reject) => {
		const t = newTransaction(store, true),
			r = t.put(obj)
		r.onerror = () =>
			reject(r.error)
		r.onsuccess = () =>
			resolve()
	})
}
