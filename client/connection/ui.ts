import lang from "../lang"
import { syncStatus } from "./state"
import { handlers, message } from "./messages"
import { truncateString } from "../util"

type syncNum = {
	active: number
	total:  number
	cinemaWatching: number
	cinemaPlaying: string
}

const syncEl = document.getElementById('sync'),
syncedCount = document.getElementById("sync-counter"),
statusCinema = document.getElementById("status-cinema")

// Render connection status indicator
export function renderStatus(status: syncStatus) {
	syncEl.textContent = lang.sync[status]
}

// Set synced IP active and total count to sync
export function renderSyncCount(sync: syncNum) {
	if (!sync) {
		return
	}
	syncedCount.textContent = `${sync.active.toString()} / ${sync.total.toString()}`
	if (sync.cinemaWatching === 0) {
		statusCinema.textContent = ""
	} else {
		if (!sync.cinemaPlaying) {
			sync.cinemaPlaying = "void"
		}
		statusCinema.textContent = `${sync.cinemaWatching.toString()} |` +
									` ${truncateString(sync.cinemaPlaying, 20)}`
		if(sync.cinemaWatching === 1) {
			statusCinema.title = lang.ui["cinemaStatusOneSpectatorTitle"] + sync.cinemaPlaying
		} else {
			statusCinema.title = sync.cinemaWatching +
				lang.ui["cinemaStatusMultipleSpectatorsTitle"] + sync.cinemaPlaying
		}
	}
}

handlers[message.syncCount] = renderSyncCount
