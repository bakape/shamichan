import * as db from "../db";
import * as state from "../state";
import lang from "../lang";
import * as thread from "./thread";
import * as options from "../options";
import * as posts from "../posts";
import * as util from "../util";
import * as boards from "./board";
import { BannerModal } from "../base";
import { urlBase } from "../ui/tab";

interface OpenThreadRecord {
	id: number;
	time: number;
}

export interface WatchedThreadRecord {
	id: number;
	postCount: number;
	unseen: number;
	board: string;
	subject: string;
	thumbnailURL?: string;
}

type ThreadPostCountDiff = {
	changed: { [id: number]: number };
	deleted: number[];
}

// Only active WatcherPanel instance
export let watcherPanel: WatcherPanel;

// Thread Watcher panel
class WatcherPanel extends BannerModal {
	constructor() {
		super(document.getElementById("watcher"));
		this.load();
	}

	public addRow(thread: WatchedThreadRecord) {
		let row = this.el.querySelector(`#t${thread.id}`);
		if (row) {
			// Thread already in table. Update status
			this.update(row, thread.unseen);
			return;
		}
		let tb = this.el.querySelector("tbody");
		let tr = tb.insertRow(-1);
		tr.setAttribute("id", `t${thread.id}`);
		for (let i = 0; i < 5; i++) {
			let tc = tr.insertCell(i);
			switch (i) {
				// Board
				case 0:
					tc.innerHTML = `<a href="../${thread.board}">/${thread.board}/</a>`;
					break;
				// Subject
				case 1:
					if (thread.unseen > 100) {
						tc.innerHTML = `<a class="thread-link" href="../all/${thread.id}">${thread.subject}</a>`;
					}
					else {
						tc.innerHTML = `<a class="thread-link" href="../all/${thread.id}?last=100">${thread.subject}</a>`;
					}
					break;
				// Status
				case 2:
					if (thread.unseen === 0) {
						tc.innerHTML = `<img class="status" title="No unseen posts" src="${urlBase}default.ico">`;
					}
					else if (thread.unseen > 0) {
						tc.innerHTML = `<img class="status" title="${thread.unseen} unseen posts" src="${urlBase}unread.ico">`;
					}
					break;
				// Mark as seen
				case 3:
					tc.innerHTML = '<a>[X]</a>';
					tc.addEventListener("click", () => {
						watchThread(thread.id, thread.postCount, 0, thread.board, thread.subject);
					});
					break;
				// Unwatch
				case 4:
					tc.innerHTML = '<a>[X]</a>';
					tc.addEventListener("click", () => {
						unwatchThread(thread.id);
						for (let el of document.querySelectorAll(".watcher-toggle")) {
							if (el.getAttribute("data-id") === String(thread.id)) {
								augmentToggle(el, false);
							}
						}
					});
					break;
			}
		}
	}

	public removeRow(id: number) {
		const row = this.el.querySelector(`#t${id}`);
		if (row) {
			row.parentElement.removeChild(row);
		}
	}

	private async update(row: Element ,unseen: number) {
		if (unseen === 0) {
			let link = (row.querySelector(".thread-link") as HTMLAnchorElement);
			if (link.href.indexOf("?") === -1) {
				link.setAttribute("href", link.href.concat("?last=100"));
			}

			let stat = (row.querySelector(".status") as HTMLImageElement);
			stat.src = `${urlBase}default.ico`;
			stat.title = "No unseen posts";
		}
		else if (unseen > 0) {
			if (unseen > 100) {
				// Remove ?last=100
				let link = (row.querySelector(".thread-link") as HTMLAnchorElement);
				link.setAttribute("href", link.href.substring(0, link.href.indexOf("?")));
			}
			let stat = (row.querySelector(".status") as HTMLImageElement);
			stat.src = `${urlBase}unread.ico`;
			stat.title = `${unseen} unseen posts`;
		}
	}

	private async load() {
		const watched = await getWatchedThreads();
		for (let id in watched) {
			if (parseInt(id) === state.page.thread) {
				// Clear unseen count, add row, and propagate change to other tabs
				watchThread(parseInt(id), watched[id].postCount, 0,
					watched[id].board, watched[id].subject);
			}
			else {
				this.addRow(watched[id]);
			} 
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
	const opened = new Set<number>();
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
		let unseen = 0;
		// Update fn-local thread record to avoid recomputing later
		watched[id].unseen = watched[id].unseen + diff.changed[id] - watched[id].postCount;
		
		if (!opened.has(id)) {
			toNotify.push(parseInt(k));
			unseen = watched[id].unseen;
		}
		
		// Update post count of watched thread
		proms.push(watchThread(id, diff.changed[id], unseen, watched[id].board,
			watched[id].subject));
	}
	for (let id of diff.deleted) {
		proms.push(unwatchThread(id));
	}

	if (options.canNotify()) {
		for (let thread of toNotify) {
			const data = watched[thread];
			const id = data.id; // Ensure heap allocation

			const opts = options.notificationOpts();
			const delta = data.unseen;
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
	watcherPanel = new WatcherPanel();

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
				p = unwatchThread(id);
			} else {
				if (state.page.thread) {
					p = watchCurrentThread();
				} else {
					const { subject, post_count, board } = boards.threads[id];
					p = watchThread(id, post_count, 0, board, subject);
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
	unseen: number, board: string, subject: string,
) {
	const data: WatchedThreadRecord = { id, postCount, unseen, board, subject };
	const p = state.posts.get(id);
	if (p && p.image) {
		data.thumbnailURL = posts.thumbPath(p.image.sha1, p.image.thumb_type);
	}

	await putExpiring("watchedThreads", id, data);

	if (watcherPanel) {
		watcherPanel.addRow(data);
	}
	propagateWatch(data);
}

// Mark current thread as watched or simply bump post count
export async function watchCurrentThread() {
	if (state.page.thread) {
		await watchThread(state.page.thread, thread.post_count,
			0, state.page.board, thread.subject);
		augmentToggle(document.querySelector(".watcher-toggle"), true);
	}
}

// Unmark thread as watched or simply bump post count
export async function unwatchThread(id: number) {
	await db.deleteObj("watchedThreads", id);
	watcherPanel.removeRow(id);
	propagateUnwatch(id);
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

// Update localStorage to trigger event that updates thread watcher panel
// in other meguca tabs. Add timestamp to "ensure" localstorage updates
function propagateWatch(thread: WatchedThreadRecord) {
	let message = { stamp: Date.now(), thread: thread, func: "watch" };
	localStorage.setItem("toggle", JSON.stringify(message));
}

function propagateUnwatch(id: number) {
	let message = { stamp: Date.now(), thread: id, func: "unwatch" };
	localStorage.setItem("toggle", JSON.stringify(message));
}

// Proxy observer for IndexedDB until that gets proper observers
// Updates thread watcher panel when threads are (un)watched in other tabs
// TODO: replace this with IndexedDB observer if/when it gets browser support
util.on(window,
	"storage",
	(e: StorageEvent) => {
		if (e.key != "toggle") {
			return;
		}
		let message = JSON.parse(e.newValue);
		switch (message.func) {
			case "watch":
				watcherPanel.addRow(message.thread);
				for (let el of document.querySelectorAll(".watcher-toggle")) {
					if (message.thread.id === Number(el.getAttribute("data-id"))) {
						augmentToggle(el, true);
					}
				}
				break;
			case "unwatch":
				for (let el of document.querySelectorAll(".watcher-toggle")) {
					if (message.thread === Number(el.getAttribute("data-id"))) {
						augmentToggle(el, false);
					}
				}
				watcherPanel.removeRow(message.thread);
				break;
		}
	});