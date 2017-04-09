import lang from "../lang"
import { syncStatus } from "./state"
import { handlers, message } from "./messages"

const syncEl = document.getElementById('sync'),
	syncedCount = document.getElementById("sync-counter")

// Render connection status indicator
export function renderStatus(status: syncStatus) {
	syncEl.textContent = lang.sync[status]
}

handlers[message.syncCount] = (n: number) =>
	syncedCount.textContent = n.toString()
