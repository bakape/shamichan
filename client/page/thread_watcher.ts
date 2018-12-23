import * as db from "../db";
import * as state from "../state";
import lang from "../lang";
import * as thread from "./thread";
import * as options from "../options";
import * as posts from "../posts";

interface OpenThreadRecord {
	id: number;
	time: number;
}

interface WatchedThreadRecord {
	id: number;
	postCount: number;
	subject: string;
	thumbnailURL?: string;
}

type ThreadPostCountDiff = {
	changed: { [id: number]: number };
	deleted: number[];
}

async function putExpiring(store: string,
	thread: number,
	data: { [key: string]: any },
) {
	data["id"] = thread;
	data["expires"] = Date.now() + 90 * 24 * 60 * 60 * 1000;
	await db.putObj(store, data).catch(console.error);
}

async function getStored(): Promise<{ [id: number]: WatchedThreadRecord }> {
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
	const watched = await getStored();
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
	for (let k in diff.changed) {
		const id = parseInt(k);

		// Update post count of watched thread
		proms.push(watchThread(id, diff.changed[id], watched[id].subject));

		if (!opened.has(id)) {
			toNotify.push(parseInt(k));
		}
	}
	for (let id of diff.deleted) {
		proms.push(unwatchThread(id));
	}


	if (options.canNotify()) {
		for (let thread of toNotify) {
			const data = watched[thread];
			const id = data.id; // Ensure heap allocation

			const opts = options.notificationOpts();
			const delta = diff.changed[id] - data.postCount;
			opts.body = lang.format["newPostsInThread"]
				.replace("%d", delta.toString());
			opts.tag = `watched_thread:${id}`;
			opts.renotify = true;
			if (options.canShowImages() && data.thumbnailURL) {
				opts.icon = data.thumbnailURL;
			}
			const n = new Notification(data.subject, opts);
			n.onclick = () => {
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
	setInterval(markThreadOpened, 1000);
	markThreadOpened();
	setInterval(fetchWatchedThreads, 60 * 1000);
	fetchWatchedThreads();
}

// Mark thread as watched
export async function watchThread(id: number, postCount: number,
	subject: string,
) {
	if (!options.canNotify()) {
		return;
	}

	const data: WatchedThreadRecord = { id, postCount, subject };
	const p = state.posts.get(id);
	if (p && p.image) {
		data.thumbnailURL = posts.thumbPath(p.image.SHA1, p.image.thumbType);
	}

	await putExpiring("watchedThreads", id, data);
}

// Mark current thread as watched or simply bump post count
export async function watchCurrentThread() {
	if (state.page.thread) {
		await watchThread(state.page.thread, thread.postCount, thread.subject);
	}
}

// Unmark thread as watched or simply bump post count
export async function unwatchThread(id: number) {
	await db.deleteObj("watchedThreads", id);
}
