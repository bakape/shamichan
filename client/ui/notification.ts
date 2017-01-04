// Desktop notifications on reply and such

import { storeSeenReply, boardConfig } from "../state"
import options from "../options"
import lang from "../lang"
import { thumbPath, Post } from "../posts"
import { repliedToMe } from "./tab"
import { read, write, scrollToAnchor } from "../util"

// Displayed, when there is no image in post
const defaultIcon = "/assets/notification-icon.png"

// Notify the user that one of their posts has been replied to
export default function notifyAboutReply(post: Post) {
	storeSeenReply(post.id)
	if (!document.hidden) {
		return
	}
	repliedToMe()

	const re = !options.notification
		|| typeof Notification !== "function"
		|| Notification.permission !== "granted"
	if (re) {
		return
	}

	let icon: string
	if (!options.hideThumbs && !options.workModeToggle) {
		if (post.image) {
			const {SHA1, thumbType} = post.image
			if (post.image.spoiler) {
				icon = '/assets/spoil/' + boardConfig.spoiler
			} else {
				icon = thumbPath(SHA1, thumbType)
			}
		} else {
			icon = defaultIcon
		}
	}
	const n = new Notification(lang.ui["quoted"], {
		icon,
		body: post.body,
		vibrate: true,
	})
	n.onclick = () => {
		n.close()
		window.focus()
		location.hash = "#p" + post.id

		// Next animation frame is delayed, when tab has no focus. Must scroll
		// only after render for there to be an element to scroll to.
		read(() =>
			write(scrollToAnchor))
	}
}
