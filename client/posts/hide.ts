// Hide posts you don't like

import { storeHidden, hidden } from "../state"
import { Post } from "./models"
import { panel } from "../options/view"
import { db, throwErr } from "../db"

// TODO: Thread hiding

// Hide a post and persist to database
export function hidePost(model: Post) {
	model.remove()
	storeHidden(model.id)
	panel.renderHidden(hidden.size)
}

// Clear all hidden posts
export function clearHidden() {
	hidden.clear()
	panel.renderHidden(0)

	const trans = db.transaction("hidden", "readwrite")
	trans.onerror = throwErr

	const req = trans.objectStore("hidden").clear()
	req.onerror = throwErr
}
