export * from "./banner"
export { default as FormView } from "./forms"
export { default as navigate } from "./history"
export { setTitle, postAdded } from "./tab"
export { default as notifyAboutReply, OverlayNotification } from "./notification"

import initKeyboard from "./keyboard"
import initTab from "./tab"
import initBanner from "./banner"
import OptionPanel from "./options"

export default () => {
	initKeyboard()
	initTab()
	initBanner()
	new OptionPanel()
}
