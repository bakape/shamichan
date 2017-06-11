import initNavigation from "./navigation"

export { extractConfigs, isBanned } from "./common"
export {
	incrementPostCount, default as renderThread, setThreadTitle
} from "./thread"
export { render as renderBoard } from "./board"

initNavigation()
