import lang from "../lang"
import { syncStatus } from "./state"
import { handlers, message } from "./messages"

type syncNum = {
	active: number
	total:  number
}

const syncEl = document.getElementById('sync'),
syncedCount = document.getElementById("sync-counter")

// Render connection status indicator
export function renderStatus(status: syncStatus) {
	syncEl.textContent = lang.sync[status]
}

// Set synced IP active and total count to sync
export function renderSyncCount(sync: syncNum) {
	syncedCount.textContent = sync ?
		`${sync.active.toString()} / ${sync.total.toString()}` : ''
}

handlers[message.syncCount] = renderSyncCount
