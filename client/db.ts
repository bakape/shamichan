// IndexedDB database controller

const dbVersion = 13;

let db: IDBDatabase

// Database has erred and all future calls should be ignored
// FF IndexedDB implementation is broken in private mode.
// See https://bugzilla.mozilla.org/show_bug.cgi?id=781982
// This helps bypass it.
let hasErred = false;

// Expiring post ID object stores
const postStores = [
	"mine",     // Posts created by this client
	"hidden",   // Posts hidden by client
	"seen",     // Replies to the user's posts that have already been seen
	"seenPost", // Posts that the user has viewed or scrolled past
];

// Expiring thread data stores
const threadStores = [
	"watchedThreads", // Threads currently watched
	"openThreads",    // Threads recently opened
];

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
			db.onversionchange = () => {
				db.close();
				if (location.reload) {
					location.reload(true); // Browser page
				} else if (self && self.close) {
					self.close(); // Service worker
				}
			};

			// Delay for quicker starts
			setTimeout(() => {
				for (let name of postStores.concat(threadStores)) {
					deleteExpired(name)
				}
			}, 10000)
		}
	})
		.catch(err => {
			hasErred = true
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
				createOPStore(db, name)
			}

			// Various miscellaneous objects
			db.createObjectStore('main', { keyPath: 'id' })
		case 4:
		case 5:
		case 6:
			// Recreate all previous post ID stores
			for (let name of postStores) {
				if (db.objectStoreNames.contains(name)) {
					db.deleteObjectStore(name);
				}
				createOPStore(db, name)
			}
		case 7:
			createExpiringStore(db, "watchedThreads");
		case 8:
			(event as any).currentTarget
				.transaction
				.objectStore("mine")
				.createIndex("id", "id");
		case 9:
			// Recreate all postStores, so that their primary key is the post ID
			for (let name of postStores) {
				db.deleteObjectStore(name);
				createOPStore(db, name);
			}
		case 10:
			// Fix possible complications after faulty upgrade
			if (!db.objectStoreNames.contains("watchedThreads")) {
				createExpiringStore(db, "watchedThreads");
			}
		case 11:
			// Reset and recreate
			db.deleteObjectStore("watchedThreads");
			createExpiringStore(db, "watchedThreads", true);

			createExpiringStore(db, "openThreads", true);
		case 12:
			// Reset and recreate
			db.deleteObjectStore("watchedThreads");
			createExpiringStore(db, "watchedThreads", true);
	}
}

function createExpiringStore(db: IDBDatabase,
	name: string,
	primaryKeyed: boolean = false,
): IDBObjectStore {
	const args = {};
	if (primaryKeyed) {
		args["keyPath"] = "id";
	}
	const s = db.createObjectStore(name, args);
	s.createIndex("expires", "expires");
	return s
}

// Expiring and with an "op" index
function createOPStore(db: IDBDatabase, name: string) {
	createExpiringStore(db, name).createIndex("op", "op")
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
	if (hasErred || !ops.length) {
		return Promise.resolve([])
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
	return readIDRange(store, s =>
		s.index("op").openCursor(op));
}

// Read a range objects and aggregate their IDs.
// store: name of objectStore
// criteria?: optional selector for the range of objects applicable
export async function readIDRange(store: string,
	criteria?: (s: IDBObjectStore) => IDBRequest,
): Promise<number[]> {
	if (hasErred) {
		return Promise.resolve([]);
	}
	return new Promise<number[]>((resolve, reject) => {
		const s = newTransaction(store, false);
		const req = criteria ? criteria(s) : s.openCursor();

		req.onerror = err =>
			reject(err);

		const ids: number[] = [];
		req.onsuccess = event => {
			const cursor = (event as any).target.result as IDBCursorWithValue;
			if (cursor) {
				ids.push(cursor.value.id);
				cursor.continue();
			} else {
				resolve(ids);
			}
		};
	});
}

// Run function for each record in store
export async function forEach<T>(store: string, fn: (data: T) => void) {
	return new Promise<void>((resolve, reject) => {
		const req = newTransaction(store, false).openCursor();

		req.onerror = err =>
			reject(err);

		req.onsuccess = event => {
			const cursor = (event as any).target.result as IDBCursorWithValue;
			if (cursor) {
				fn(cursor.value);
				cursor.continue();
			} else {
				resolve();
			}
		};
	});
}

// Asynchronously insert a new expiring post id object into a postStore
export function storeID(store: string, expiry: number, ...items: {id: number; op: number}[]) {
	if (hasErred) {
		return;
	}

	const expires = Date.now() + expiry;
	putAll(store, items.map(item => {
		const obj: any = Object.assign({}, item);
		obj.expires = expires;
		return { obj: item, key: item.id }
	}));
}

// Clear the target object store asynchronously
export function clearStore(store: string) {
	if (hasErred) {
		return
	}
	const trans = newTransaction(store, true),
		req = trans.clear()
	req.onerror = throwErr
}

// Retrieve an object from a specific object store
export function getObj<T>(store: string, id: any): Promise<T> {
	if (hasErred) {
		throw new Error("IndexedDB not accessible");
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
export function putObj(store: string, obj: any, key: any = undefined,
): Promise<void> {
	if (hasErred) {
		return Promise.resolve(undefined)
	}
	return new Promise<void>((resolve, reject) => {
		const t = newTransaction(store, true),
			r = t.put(obj, key)
		r.onerror = () =>
			reject(r.error)
		r.onsuccess = () =>
			resolve()
	})
}

export function putAll(store: string, toAdd: {obj: any, key?: any}[]): Promise<void> {
	if (hasErred) {
		return Promise.resolve(undefined)
	}

	return new Promise<void>((resolve, reject) => {
		const objStore = newTransaction(store, true), transaction = objStore.transaction;

		for (const {obj, key} of toAdd) {
			objStore.put(obj, key);
		}

		transaction.oncomplete = () => resolve();
		transaction.onerror = () => reject(transaction.error);
	});
}

// Delete an object from a store by ID
export function deleteObj(store: string, id: number): Promise<void> {
	if (hasErred) {
		return Promise.resolve(undefined);
	}
	return new Promise<void>((resolve, reject) => {
		const t = newTransaction(store, true);
		const r = t.delete(id);
		r.onerror = () =>
			reject(r.error);
		r.onsuccess = () =>
			resolve();
	});
}
