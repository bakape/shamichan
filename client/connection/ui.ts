import lang from "../lang"
import { syncStatus } from "./state"
import { handlers, message } from "./messages"

type syncNum = {
	clients: number
	idle:    number
}

const syncEl = document.getElementById('sync'),
syncedCount = document.getElementById("sync-counter")

// Render connection status indicator
export function renderStatus(status: syncStatus) {
	syncEl.textContent = lang.sync[status]
}

// Set synced IP count and idle count to sync
export function renderSyncCount(sync: syncNum) {
	syncedCount.textContent = sync ?
		`${sync.clients.toString()} / ${sync.idle.toString()}` : ""
}

handlers[message.syncCount] = renderSyncCount
