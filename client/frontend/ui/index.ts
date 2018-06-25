export * from "./banner"
export { default as FormView } from "./forms"
export { postAdded } from "./tab"
export { notifyAboutReply, OverlayNotification } from "./notification"
export { default as CaptchaView } from "./captcha"

import initKeyboard from "./keyboard"
import initTab from "./tab"
import initBanner from "./banner"
import OptionPanel from "./options"
import { initNotifications } from "./notification";

export default () => {
	initKeyboard()
	initTab()
	initBanner()
	initNotifications()
	new OptionPanel()
}
