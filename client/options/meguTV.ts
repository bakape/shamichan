import options from ".";
import { setAttrs } from "../util";
import { page } from "../state";
import { sourcePath, serverNow } from "../posts";
import { fileTypes } from "../common"
import { handlers, message, connSM, connState, send } from "../connection"

type Data = {
	elapsed: number;
	playlist: Video[]
};

type Video = {
	sha1: string
	fileType: fileTypes
};

let playlist: Video[];
let lastStart = 0;

function render() {
	if (!playlist) {
		return
	}

	let cont = document.getElementById("megu-tv")
	if (!cont) {
		cont = document.createElement("div")
		setAttrs(cont, {
			id: "megu-tv",
			class: "modal glass",
			style: "display: block;",
		});
		document.getElementById("modal-overlay").prepend(cont);
	}

	// Remove old videos and add new ones, while preserving existing one.
	// Should help caching.
	const existing: { [sha1: string]: HTMLVideoElement } = {};
	for (let ch of [...cont.children] as HTMLVideoElement[]) {
		ch.pause();
		ch.remove();
		existing[ch.getAttribute("data-sha1")] = ch;
	}
	for (let i = 0; i < playlist.length; i++) {
		let el = existing[playlist[i].sha1];
		if (!el) {
			el = document.createElement("video");
			el.setAttribute("data-sha1", playlist[i].sha1);
			el.setAttribute("style", "max-width:30vw");
			el.controls = true;
			el.src = sourcePath(playlist[i].sha1, playlist[i].fileType);
			el.volume = options.audioVolume / 100;
		}

		// Buffer videos about to play by playing them hidden and muted
		if (!i) {
			el.currentTime = serverNow() - lastStart;
			el.classList.remove("hidden");
			el.muted = false;
		} else {
			el.muted = true;
			el.classList.add("hidden");
		}
		cont.append(el);
		el.play();
	}
}

export function persistMessages() {
	handlers[message.meguTV] = (data: Data) => {
		lastStart = serverNow() - data.elapsed;
		playlist = data.playlist;
		if (options.meguTV) {
			render();
		}
	}

	// Subscribe to feed on connection
	connSM.on(connState.synced, subscribe);
}

function subscribe() {
	if (options.meguTV) {
		send(message.meguTV, null);
	}
}

export default function () {
	const el = document.getElementById("megu-tv");
	if (el || page.board === "all" || !page.thread) {
		return;
	}
	if (connSM.state === connState.synced) {
		subscribe();
	}
	render();

	// Handle toggling of the option
	options.onChange("meguTV", on => {
		if (on && page.board !== "all") {
			if (!document.getElementById("megu-tv")) {
				render();
			}
		} else {
			const el = document.getElementById("megu-tv");
			if (el) {
				el.remove();
			}
		}
	});
}
