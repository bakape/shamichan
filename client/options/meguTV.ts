import options from ".";
import { HTML, makeFrag } from "../util";
import { page } from "../state";
import { sourcePath } from "../posts";
import { fileTypes } from "../common"

function render() {
	const html = HTML
		`<div id=megu-tv class="modal glass" style="display: block;">
			<video id=megu-tv-player controls style="max-width:30vw"></video>
		</div>`;
	document.getElementById("modal-overlay").prepend(makeFrag(html));

	const el = document.getElementById("megu-tv-player") as HTMLVideoElement;
	el.onerror = el.onended = el.onclick = setSource;
	setSource();
}

async function setSource() {
	const el = document.getElementById("megu-tv-player") as HTMLVideoElement;
	if (!el) {
		return;
	}
	const res = await fetch(`/api/random-video/${page.board}`);
	if (res.status !== 200) {
		throw "video not found: ${await res.text()}";
	}
	el.src = sourcePath(await res.text(), fileTypes.webm);
	el.currentTime = 0;
	el.play();
}

export default function () {
	const el = document.getElementById("megu-tv");
	if (el || page.board === "all") {
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
	})
}
