// TEMP
export default null

import {HTML, on, makeEl} from '../util'
import {$threads} from '../page/common'
import View from '../view'
import Model from '../model'
import {write} from '../render'
import {FormView} from '../forms'
import {Captcha} from '../captcha'

interface ThreadCreationRequest extends Captcha {
	name: string
	email: string
	auth: string
	password: string
	subject: string
	board: string
}

// For ensuring we have unique captcha IDs
let threadFormCounter = 0

on($threads, "click", e => new ThreadForm(e), {selector: ".new-thread-button"})

// Form view for creating new threads
class ThreadForm extends FormView {
	$parent: Element
	$aside: Element

	constructor(event: Event) {
		super({}, () =>
			this.sendRequest())
		this.$parent = event.target as Element
		this.$aside = this.$parent.closest("aside")
		this.render()
	}

	// Render the element
	render() {
		const html = ""
		this.renderForm(html)
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

	sendRequest() {

	}
}
