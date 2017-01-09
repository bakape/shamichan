// Hide posts you don't like

import { storeHidden, hidden } from "../state"
import { Post } from "./model"
import { clearStore } from "../db"
import { trigger } from "../util"

// TODO: Thread hiding

// Hide a post and persist to database
export function hidePost(model: Post) {
	model.remove()
	storeHidden(model.id)
	trigger("renderHiddenCount", hidden.size)
}

// Clear all hidden posts
export function clearHidden() {
	hidden.clear()
	trigger("renderHiddenCount", 0)
	clearStore("hidden")
}
