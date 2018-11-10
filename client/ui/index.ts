export * from "./banner"
export { default as FormView, FormAttrs } from "./forms"
export { postAdded } from "./tab"
export { default as notifyAboutReply, OverlayNotification } from "./notification"
export { renderCaptchaForm, captchaLoaded } from "./captcha"

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
