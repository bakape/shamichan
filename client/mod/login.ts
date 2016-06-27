// Login/logout facilities for the account system

import {TabbedModal} from '../banner'
import {write} from '../render'
import {defer} from '../defer'
import {mod as lang} from '../lang'
import {setLabel} from '../util'

// Account login and registration
class AccountPanel extends TabbedModal {
	$login: HTMLFormElement
	$register: HTMLFormElement

	constructor() {
		super({el: document.querySelector('#account-panel')})
		this.$login = this.el
			.querySelector("#login-form") as HTMLFormElement
		this.$register = this.el
			.querySelector("#registration-form") as HTMLFormElement
		write(() => this.renderInitial())
	}

	// Render localised labels to the login and registration forms
	renderInitial() {
		const {el} = this,
			tabLinks = el.querySelectorAll(".tab-link")
		tabLinks[0].textContent = lang.login
		tabLinks[1].textContent = lang.register

		for (let tab of [this.$login, this.$register]) {
			for (let name of ["login", "password"]) {
				setLabel(tab, name, lang[name])
			}
			(tab.lastChild as HTMLInputElement).value = lang.submit
		}

		setLabel(el, "repeat", lang.repeat)
	}
}

defer(() => new AccountPanel)
