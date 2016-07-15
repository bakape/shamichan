import View from './view'
import Model from './model'
import {write} from './render'
import {config} from './state'

// Data of a captcha challenge
export interface Captcha {
	captcha: string
	captchaID: string
}

// Wrapper around Solve Media's captcha service
export default class CaptchaView extends View<Model> {
	widget: ACPuzzleController
	id: string

	constructor(id: string) {
		super({id: id})
		this.render()
	}

	render() {
		this.widget = ACPuzzle.create(config.captchaPublicKey, this.id, {
			multi: true
		})
	}

	remove() {
		write(() =>
			this.widget.destroy())
		super.remove()
	}

	// Returns the data from the captcha widget
	data(): Captcha {
		return {
			captcha: ACPuzzle.get_response(),
			captchaID: ACPuzzle.get_challenge(),
		}
	}
}
