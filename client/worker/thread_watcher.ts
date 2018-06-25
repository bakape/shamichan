import { putObj, readIDRange, deleteObj, getObj } from "../common/db"
import { forward } from "./ipc"
import { WatcherMessage } from "../common/ipc"
import { LanguagePack } from "../common/types";

declare var lang: LanguagePack;

let es: EventSource;
let threads = new Set<number>();

// Start or restart watching for threads. The threads to be watched are read
// from IndexedDB.
export async function watch() {
    threads = new Set(await readIDRange("watchedThreads"));
    if (es) {
        es.close();
        es = null;
    }

    let url = "/json/watch?";
    let first = true;
    for (let id of threads) {
        if (first) {
            first = false;
        } else {
            url += "&";
        }
        url += `id=${id}`;
    }

    es = new EventSource(url);
    es.onmessage = onMessage;
}


async function onMessage(e: MessageEvent) {
    const msg: WatcherMessage = JSON.parse(e.data);
    const mine = new Set<number>();
    if (msg.links) {
        for (let { id } of msg.links) {
            if (await getObj<any>("mine", id)) {
                msg.repliedToMe = true;
                mine.add(id);
            }
        }
    }
    msg.body = addYous(msg.body, lang.posts["you"], mine);

    await forward(msg.op, msg, () => {
        if (!mine.size
            || (Notification as any).permission !== "granted"
        ) {
            return;
        }
        ((self as any).registration as any).showNotification(
            lang.ui["quoted"],
            {
                body: msg.body,
                icon: msg.image || "/assets/notification-icon.png",
                requireInteraction: true,
                vibrate: [500],
                data: msg.id,
            },
        );
    });
}

// Add or refresh a thread to the watched list and restart watching
export async function watchThread(id: number, lastSeen: number) {
    await putObj(
        "watchedThreads",
        {
            id, lastSeen,
            expires: Date.now() + 30 * 24 * 60 * 60 * 1000,
        },
        id, );

    // Skip restarting, if thread is already watched
    if (!threads.has(id)) {
        await watch();
    }
}

// Remove a thread from the watched list
export async function unwatchThread(id: number) {
    await deleteObj("watchedThreads", id);

    // Skip restarting, if thread is already unwatched
    if (threads.has(id)) {
        await watch();
    }
}


// Parse body and add (You)s
function addYous(body: string, you: string, mine: Set<number>): string {
    for (let id of mine) {
        body = body.replace(new RegExp(">>" + id, "g"), `>>${id} ${you}`);
    }
    return body;
}

