// Login/logout facilities for the account system

import {TabbedModal} from '../banner'
import {write} from '../render'
import {defer} from '../defer'
import {mod as lang} from '../lang'
import {setPlaceholder} from '../util'

// Account login and registration
class AccountPanel extends TabbedModal {
	constructor() {
		super({el: document.querySelector('#account-panel')})
		write(() => this.render())
	}

	// Render localised labels to the login and registration forms
	render() {
		const {el} = this,
			tabLinks = el.querySelectorAll(".tab-link")
		tabLinks[0].textContent = lang.login
		tabLinks[1].textContent = lang.register

		for (let sel of ["#login-form", "#registration-form"]) {
			const tab = el.querySelector(sel)
			for (let sel of ["login", "password"]) {
				setPlaceholder(tab, `input[name=${sel}]`,  lang[sel])
			}
			(tab.lastChild as HTMLInputElement).value = lang.submit
		}

		setPlaceholder(el, "input[name=password-repeat]", lang.repeat)
	}
}

defer(() => new AccountPanel)
