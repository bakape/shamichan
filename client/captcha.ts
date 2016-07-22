import View from './view'
import Model from './model'
import {write} from './render'
import {config} from './state'

// Data of a captcha challenge
export interface Captcha {
	captcha: string
	captchaID: string
}

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<Model> {
	widget: ACPuzzleController
	id: string

	constructor(id: string) {
		super({el: document.getElementById(id)})
		this.render()
	}

	render() {
		this.widget = ACPuzzle.create(config.captchaPublicKey, this.id, {
			multi: true,
		})
	}

	// Load a new captcha
	reload() {
		this.widget.reload()
	}

	remove() {
		write(() =>
			this.widget.destroy())
		super.remove()
	}

	// Returns the data from the captcha widget
	data(): Captcha {
		return {
			captcha: this.widget.get_response(),
			captchaID: this.widget.get_challenge(),
		}
	}
}
