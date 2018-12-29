import { storeSeenReply, seenReplies } from "../state"
import * as options from "../options";
import lang from "../lang"
import { thumbPath, Post } from "../posts"
import { repliedToMe } from "./tab"
import * as util from "../util"
import { View } from "../base"

// Notify the user that one of their posts has been replied to
export default function notifyAboutReply(post: Post) {
	if (seenReplies.has(post.id)) {
		return
	}
	storeSeenReply(post.id, post.op)
	if (!document.hidden) {
		return
	}
	repliedToMe(post)

	if (!options.canNotify()) {
		return
	}

	const opts = options.notificationOpts();
	if (options.canShowImages() && post.image) {
		const { SHA1, thumbType, spoiler } = post.image;
		if (spoiler) {
			opts.icon = '/assets/spoil/default.jpg';
		} else {
			opts.icon = thumbPath(SHA1, thumbType);
		}
	}
	opts.body = post.body;
	const n = new Notification(lang.ui["quoted"], opts)
	n.onclick = () => {
		n.close()
		window.focus()
		location.hash = "#p" + post.id
		util.scrollToAnchor()
	}
}

// Textual notification at the top of the page
export class OverlayNotification extends View<null> {
	constructor(text: string) {
		super({
			el: util.importTemplate("notification")
				.firstChild as HTMLElement,
		})
		this.on("click", () =>
			this.remove())
		this.el.querySelector("b").textContent = text
		document.getElementById("modal-overlay").prepend(this.el)
	}
}
