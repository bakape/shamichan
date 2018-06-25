import { open } from "../common/db"
import {
	Message, Type, WatchThreadRequest, UnwatchThreadRequest,
} from "../common/ipc"
import { watch, watchThread, unwatchThread } from "./thread_watcher"

// Start service worker
export async function start() {
	await open();
	watch();
}

// Handle messages from client
export function onMessage(e: MessageEvent) {
	const data = e.data as Message;
	switch (data.type) {
		case Type.watchThread:
			{
				const { thread, lastSeen } = data as WatchThreadRequest;
				watchThread(thread, lastSeen);
			}
			break;
		case Type.unwatchThread:
			unwatchThread((data as UnwatchThreadRequest).thread);
			break;
	}
}
