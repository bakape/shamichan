import { storeSeenReply, seenReplies } from "../state"
import options from "../options"
import lang from "../lang"
import { thumbPath, Post } from "../posts"
import { repliedToMe } from "./tab"
import { scrollToAnchor, importTemplate } from "../util"
import { View } from "../base"

// Displayed, when there is no image in post
const defaultIcon = "/assets/notification-icon.png",
	overlay = document.getElementById("modal-overlay")

// Notify the user that one of their posts has been replied to
export default function notifyAboutReply(post: Post) {
	if (seenReplies.has(post.id)) {
		return
	}
	storeSeenReply(post.id)
	if (!document.hidden) {
		return
	}
	repliedToMe()

	const re = !options.notification
		|| typeof Notification !== "function"
		|| (Notification as any).permission !== "granted"
	if (re) {
		return
	}

	let icon: string
	if (!options.hideThumbs && !options.workModeToggle) {
		if (post.image) {
			const { SHA1, thumbType } = post.image
			if (post.image.spoiler) {
				icon = '/assets/spoil/default.jpg'
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
		scrollToAnchor()
	}
}

// Textual notification at the top of the page
export class OverlayNotification extends View<null> {
	constructor(text: string) {
		super({ el: importTemplate("notification").firstChild as HTMLElement })
		this.on("click", () =>
			this.remove())
		this.el.querySelector("b").textContent = text
		overlay.prepend(this.el)
	}
}
