/*
 IndexedDB manager
*/

let db :IDBDatabase

// Execute a database request as a promise
export function exec(request: IDBRequest): Promise<any> {
	return new Promise((resolve, reject) => {
		request.onerror = () => reject(request.error)
		request.onsuccess = () => resolve(request.result)
	})
}

export async function open() {
	db = await exec(indexedDB.open('meguca'))
}
