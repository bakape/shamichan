// TEMP
export default null

import {HTML, on, makeEl} from '../util'
import {$threads} from '../page/common'
import View from '../view'
import Model from '../model'
import {write} from '../render'
import CaptchaView, {Captcha} from '../captcha'

interface ThreadCreationRequest extends Captcha {
	name: string
	email: string
	auth: string
	password: string
	subject: string
	board: string
}

on($threads, "click", e => new ThreadForm(e), {selector: ".new-thread-button"})

// Form view for creating new threads
class ThreadForm extends View<ThreadCreationRequest> {
	$parent: Element
	$aside: Element

	constructor(event: Event) {
		super({
			tag: "form",
			cls: "thread-form",
		})
		this.$parent = event.target as Element
		this.$aside = this.$parent.closest("aside")
		this.render()
	}

	render() {
		const html = HTML
			`<input type="text" name="subject">
			<label for="subject">`
		this.el.innerHTML = html
		write(() => {
			this.$parent.style.display = "none"
			this.$aside.classList.remove("act")
			this.$aside.append(this.el)
		})
	}

	remove() {
		super.remove()
		write(() => {
			this.$parent.style.display = ""
			this.$aside.classList.add("act")
		})
	}
}
