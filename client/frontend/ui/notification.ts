import { storeSeenReply, seenReplies } from "../state"
import { Post } from "../posts"
import { repliedToMe } from "./tab"
import { scrollToAnchor, importTemplate } from "../util"
import { View } from "../base"
import { WatcherMessage } from "../../common/ipc";
import lang from "../lang";

// Notify the user that one of their posts has been replied to
export function notifyAboutReply(post: Post) {
	if (seenReplies.has(post.id)) {
		return
	}
	storeSeenReply(post.id, post.op)
	repliedToMe(post)
}

// Listen for service worker notifications
export function initNotifications() {
	if (!navigator.serviceWorker) {
		return;
	}
	navigator.serviceWorker.addEventListener('message', e => {
		if (!document.hidden
			|| (Notification as any).permission !== "granted"
		) {
			return;
		}

		const msg = e.data as WatcherMessage;
		const n = new Notification(lang.ui["quoted"], {
			body: msg.body,
			icon: msg.image || "/assets/notification-icon.png",
			requireInteraction: true,
			vibrate: [500],
			data: msg.id,
		} as any);
		n.onclick = () => {
			n.close()
			window.focus()
			location.hash = "#p" + msg.id
			scrollToAnchor()
		};
	});
}

// Textual notification at the top of the page
export class OverlayNotification extends View<null> {
	constructor(text: string) {
		super({ el: importTemplate("notification").firstChild as HTMLElement })
		this.on("click", () =>
			this.remove())
		this.el.querySelector("b").textContent = text
		document.getElementById("modal-overlay").prepend(this.el)
	}
}
