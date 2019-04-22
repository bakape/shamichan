import * as db from "../db";
import * as state from "../state";
import lang from "../lang";
import * as thread from "./thread";
import * as options from "../options";
import * as posts from "../posts";
import * as util from "../util";
import * as boards from "./board";
import {BannerModal} from "../base";
import {urlBase} from "../ui/tab";

interface OpenThreadRecord {
	id: number;
	time: number;
}

export interface WatchedThreadRecord {
	id: number;
	postCount: number;
	board: string;
	subject: string;
	thumbnailURL?: string;
}

type ThreadPostCountDiff = {
	changed: { [id: number]: number };
	deleted: number[];
}

// Only active WatcherPanel instance
export let watcherPanel: WatcherPanel

// Thread Watcher panel
class WatcherPanel extends BannerModal {
	constructor() {
		super(document.getElementById("watcher"))
		this.load()
	}
	
	public addRow(thread: WatchedThreadRecord) {
		let tb = <HTMLTableElement> this.el.firstElementChild.lastElementChild
		let tr = tb.insertRow(-1)
		tr.setAttribute("id", String(thread.id))
		for (let i = 0; i < 4; i++) {
			let tc = tr.insertCell(i)
			switch (i) {
				// Board
				case 0: {
					tc.innerHTML = `<a href="../${thread.board}">/${thread.board}/</a>`
					break
				}
				// Subject
				case 1: {
					tc.innerHTML = `<a href="../all/${thread.id}">${thread.subject}</a>`
					break
				}
				// Status
				case 2: {
					tc.innerHTML = `<img id="status" src="${urlBase}default.ico">`
					break
				}
				// Unwatch
				case 3: {
					tc.innerHTML = `<a>[X]</a>`
					tc.addEventListener("click", () => {
						this.removeRow(thread.id)
						unwatchThread(thread.id)
						for ( let el of document.querySelectorAll(".watcher-toggle")) {
							if ( el.getAttribute("data-id") === String(thread.id) ) {
								augmentToggle(el, false)
							}
						}
					})
					break
				}
			}
		}
	}

	public removeRow(id: Number) {
		let row = document.getElementById(String(id))
		row.parentElement.removeChild(row)
	}

	private async load() {
		const watched = await getWatchedThreads()
		for (let id in watched) {
			this.addRow(watched[id])
		}
	}
}

async function putExpiring(store: string,
	thread: number,
	data: { [key: string]: any },
) {
	data["id"] = thread;
	data["expires"] = Date.now() + 90 * 24 * 60 * 60 * 1000;
	await db.putObj(store, data).catch(console.error);
}

// Return all watched threads
export async function getWatchedThreads()
	: Promise<{ [id: number]: WatchedThreadRecord }> {
	const watched = {};
	await db.forEach<WatchedThreadRecord>("watchedThreads", rec =>
		watched[rec.id] = rec);
	return watched;
}

// Return set of opened threads across all tabs
async function getOpenedThreads(): Promise<Set<number>> {
	const opened = new Set();
	await db.forEach<OpenThreadRecord>("openThreads", ({ id, time }) => {
		// Use 3 times the write interval to account for some latency.
		if (time >= Date.now() - 3 * 1000) {
			opened.add(id);
		}
	});
	return opened;
}

async function fetchWatchedThreads() {
	const last = localStorage.getItem("last_watched_fetched");
	if (last && parseInt(last) > Date.now() - 60 * 1000) {
		return;
	}
	const watched = await getWatchedThreads();
	if (!Object.keys(watched).length) {
		return;
	}
	// Minimize chance of multiple concurrent checks
	localStorage.setItem("last_watched_fetched", Date.now().toString());

	const body = {};
	for (let id in watched) {
		body[id] = watched[id].postCount;
	}
	const res = await fetch("/json/thread-updates", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify(body),
	});
	if (res.status != 200) {
		throw Error("watched threads: " + await res.text());
	}

	const diff: ThreadPostCountDiff = await res.json();
	const proms = [];
	const toNotify = [];
	const opened = await getOpenedThreads();
	if (state.page.thread) {
		// Accounts for some latency between the DB
		opened.add(state.page.thread);
	}
	for (let k in diff.changed) {
		const id = parseInt(k);

		// Update post count of watched thread
		proms.push(watchThread(id, diff.changed[id], watched[id].board, watched[id].subject));

		if (!opened.has(id)) {
			toNotify.push(parseInt(k));
		}
	}
	for (let id of diff.deleted) {
		watcherPanel.removeRow(id);
		proms.push(unwatchThread(id));
	}

	if (options.canNotify()) {
		for (let thread of toNotify) {
			const data = watched[thread];
			const id = data.id; // Ensure heap allocation

			const opts = options.notificationOpts();
			const delta = diff.changed[id] - data.postCount;
			opts.body = `/${data.board}/ - 「${data.subject}」`

			// Persist target, even if browser tab closed
			opts.data = { id, delta };

			opts.tag = `watched_thread:${id}`;
			opts.renotify = true;
			if (options.canShowImages() && data.thumbnailURL) {
				opts.icon = data.thumbnailURL;
			}
			const n = new Notification(
				lang.format["newPostsInThread"]
					.replace("%d", delta.toString()),
				opts);
			n.onclick = function () {
				const { id, delta } = this.data;
				let u = `/all/${id}`;
				if (delta <= 100) {
					u += "?last=100";
				}
				window.open(u);
			};
		}
	}

	return await Promise.all(proms);
}

function markThreadOpened() {
	if (!state.page.thread) {
		return;
	}
	putExpiring("openThreads", state.page.thread, {
		time: Date.now(),
	});
}

export function init() {
	watcherPanel = new WatcherPanel()

	setInterval(markThreadOpened, 1000);
	markThreadOpened();

	setInterval(fetchWatchedThreads, 60 * 1000);
	fetchWatchedThreads();

	localizeThreadWatchToggles();

	// Handle toggle clicks
	util.on(document,
		"click",
		(e: MouseEvent) => {
			if (e.which != 1) {
				return;
			}
			const el = (e.target as Element).closest(".watcher-toggle");
			const id = parseInt(el.getAttribute("data-id"));
			let p;
			if (el.classList.contains("enabled")) {
				augmentToggle(el, false);
				watcherPanel.removeRow(id);
				p = unwatchThread(id);
			} else {
				if (state.page.thread) {
					p = watchCurrentThread();
				} else {
					const { subject, postCtr, board } = boards.threads[id];
					p = watchThread(id, postCtr, board, subject);
				}
				augmentToggle(el, true);
			}
			p.catch(console.error);
		},
		{
			selector:
				".watcher-toggle, .watcher-toggle svg, .watcher-toggle path",
			passive: true,
		});
}

// Mark thread as watched
export async function watchThread(id: number, postCount: number,
	board: string, subject: string,
) {
	if (!options.canNotify()) {
		return;
	}

	const data: WatchedThreadRecord = { id, postCount, board, subject };
	const p = state.posts.get(id);
	if (p && p.image) {
		data.thumbnailURL = posts.thumbPath(p.image.sha1, p.image.thumb_type);
	}

	await putExpiring("watchedThreads", id, data);

	watcherPanel.addRow(data)
}

// Mark current thread as watched or simply bump post count
export async function watchCurrentThread() {
	if (state.page.thread) {
		await watchThread(state.page.thread, thread.postCount, state.page.board ,thread.subject);
		augmentToggle(document.querySelector(".watcher-toggle"), true);
	}
}

// Unmark thread as watched or simply bump post count
export async function unwatchThread(id: number) {
	await db.deleteObj("watchedThreads", id);
}

// Toggle all thread watching buttons according to DB state
async function localizeThreadWatchToggles() {
	const watched = new Set(Object.keys(await getWatchedThreads()));
	for (let el of document.querySelectorAll(".watcher-toggle")) {
		if (watched.has(el.getAttribute("data-id"))) {
			augmentToggle(el, true);
		}
	}
}

// Augment thread watching toggle
function augmentToggle(el: Element, enabled: boolean) {
	el.classList.toggle("enabled", enabled);
	el.setAttribute("title",
		lang.ui[enabled ? "unwatchThread" : "watchThread"]);
}
