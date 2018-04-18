import options from ".";
import { HTML, makeFrag } from "../util";
import { page } from "../state";
import { sourcePath } from "../posts";
import { fileTypes } from "../common"
import { handlers, message } from "../connection"

type Video = {
	elapsed: number;
	sha1: string
};

let sha1 = "";
let lastStart = 0;

function render() {
	let el = document.getElementById("megu-tv-player") as HTMLVideoElement;
	if (!el) {
		const html = HTML
			`<div id=megu-tv class="modal glass" style="display: block;">
				<video id=megu-tv-player controls style="max-width:30vw"></video>
			</div>`;
		document.getElementById("modal-overlay").prepend(makeFrag(html));
		el = document.getElementById("megu-tv-player") as HTMLVideoElement;
	}

	if (sha1) {
		el.src = sourcePath(sha1, fileTypes.webm);
		el.currentTime = Math.floor(Date.now() / 1000) - lastStart;
		el.play();
	}
}

export function persistMessages() {
	handlers[message.meguTV] = (data: Video) => {
		sha1 = data.sha1;
		lastStart = Math.floor(Date.now() / 1000) - data.elapsed;
		if (options.meguTV) {
			render();
		}
	}
}

export default function () {
	const el = document.getElementById("megu-tv");
	if (el || page.board === "all" || !page.thread) {
		return;
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
