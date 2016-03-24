interface IDBRequest {
	exec(): Promise<any>
}

type IDBTransactionMode = 'readonly' | 'readwrite' | 'versionchange'

interface IDBDatabase {
	transaction(scope: string|string[], transactionMode?: IDBTransactionMode): IDBTransaction
}
