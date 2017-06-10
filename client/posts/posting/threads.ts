import { on } from '../../util'
import { CaptchaView } from "../../ui"

function expand(e: Event) {
	const el = (e.target as HTMLElement).closest("aside")
	el.classList.add("expanded")
	const c = el.querySelector(".captcha-container")
	if (c) {
		new CaptchaView(c)
	}
}

export default () =>
	on(document.getElementById("threads"), "click", expand, {
		selector: ".new-thread-button",
		passive: true,
	})
