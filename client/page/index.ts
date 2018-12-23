import initNavigation from "./navigation"
import * as watcher from "./thread_watcher";

export { extractConfigs } from "./common"
export { incrementPostCount, default as renderThread } from "./thread"
export { render as renderBoard } from "./board"
export { watchCurrentThread } from "./thread_watcher";

export function init() {
	initNavigation();
	watcher.init();
}
