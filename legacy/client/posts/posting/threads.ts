import { on, scrollToElement } from '../../util'

function expand(e: Event) {
	const el = (e.target as HTMLElement).closest("aside")
	el.classList.add("expanded");
	const c = el.querySelector(".captcha-container") as HTMLElement
	if (c) {
		const ns = c.querySelector("noscript");
		if (ns) {
			c.innerHTML = ns.innerHTML;
		}
	}
}

// Manually expand thread creation form, if any
export function expandThreadForm() {
	const tf = document.querySelector("aside:not(.expanded) .new-thread-button") as HTMLElement
	if (tf) {
		tf.click()
		scrollToElement(tf)
	}
}

export default () =>
	on(document.getElementById("threads"), "click", expand, {
		selector: ".new-thread-button",
		passive: true,
	})
