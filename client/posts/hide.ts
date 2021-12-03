// Hide posts you don't like

import { storeHidden, hidden, posts, mine, page } from "../state"
import { Post } from "./model"
import { clearStore } from "../db"
import { trigger } from "../util"
import options from "../options"

// Hide a post and persist to database
export function hidePost(model: Post) {
	hideRecursively(model)
	storeHidden(model.id, model.op)
	trigger("renderHiddenCount", hidden.size)
}

// Clear all hidden posts
export function clearHidden() {
	hidden.clear()
	trigger("renderHiddenCount", 0)
	clearStore("hidden")
	for (let p of posts) {
		p.unhide()
		if (p.id == p.op) {
			if (page.catalog) {
				document.getElementById(`p${p.id}`).classList.remove("hidden")
			} else {
				// Unhide thread from board index
				document.querySelector(`section[data-id="${p.id}"]`).classList.remove("hidden")
			}
		}
	}
}

// Hide all posts that reply to post recursively
export function hideRecursively(post: Post) {
	if (post.hidden || mine.has(post.id)) {
		return
	}
	post.hide()

	// Also add posts linking hidden posts recursively to hidden post set, but
	// don't persist these.
	hidden.add(post.id)

	if (post.backlinks && options.hideRecursively) {
		for (let id in post.backlinks) {
			const p = posts.get(parseInt(id))
			if (p) {
				hideRecursively(p)
			}
		}
	}
}
