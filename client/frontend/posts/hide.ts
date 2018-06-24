// Hide posts you don't like

import { storeHidden, hidden, posts } from "../state"
import { Post } from "./model"
import { clearStore } from "../../common/db"
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
	}
}

// Hide all posts that reply to post recursively
export function hideRecursively(post: Post) {
	if (post.hidden) {
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

	// Also hide all replies, if OP hidden
	if (post.id === post.op) {
		for (let p of posts) {
			if (p.op === post.id) {
				hideRecursively(p)
			}
		}
	}
}
