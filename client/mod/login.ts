// Login/logout facilities for the account system

import {TabbedModal} from '../banner'
import {write} from '../render'
import {defer} from '../defer'
import {mod as lang} from '../lang'
import {setLabel, on, HTML} from '../util'
import {register, send, message} from '../connection'

// Login/Registration request sent to the server through websocket
type LoginRequest = {
	id: string
	password: string
}

// Response codes of LoginResponse
const enum responseCode {
	success,
	nameTaken,
	wrongCredentials
}

// Login/Registration response received from the server
type LoginResponse = {
	code: responseCode
	session: string // Session ID token
}

let loginID = localStorage.getItem("loginID"),
	sessionToken = localStorage.getItem("sessionToken"),
	panel: AccountPanel

// Account login and registration
class AccountPanel extends TabbedModal {
	$login: HTMLFormElement = (this.el
		.querySelector("#login-form") as HTMLFormElement)
	$register: HTMLFormElement = (this.el
		.querySelector("#registration-form") as HTMLFormElement)

	constructor() {
		super({el: document.querySelector('#account-panel')})

		on(this.$register, 'submit', e => this.register(e))
		on(this.$login, 'submit', e => this.login(e))
		this.validatePasswordMatch()
		write(() => this.renderInitial())
		this.onClick({
			'#logout': () => this.logout(),
		})
	}

	// Render localised labels to the login and registration forms
	renderInitial() {
		const {el} = this,
			tabLinks = el.querySelectorAll(".tab-link")
		tabLinks[0].textContent = lang.id
		tabLinks[1].textContent = lang.register

		for (let tab of [this.$login, this.$register]) {
			for (let name of ["id", "password"]) {
				setLabel(tab, name, lang[name])
			}
			(tab.lastChild as HTMLInputElement).value = lang.submit
		}

		setLabel(el, "repeat", lang.repeat)
	}

	// Handle login form
	login(event: Event) {
		event.preventDefault()
		sendRequest(event.target as HTMLFormElement, message.login)
	}

	// Handle registration form
	register(event: Event) {
		event.preventDefault()
		sendRequest(event.target as HTMLFormElement, message.register)
	}

	// Assert passwords are equel
	validatePasswordMatch() {
		const el = this.$register,
			password = el.querySelector("input[name=password]"),
			repeat = el.querySelector("input[name=repeat]") as HTMLInputElement
		repeat.onchange = () => {
			if (repeat.value !== password.value) {
				repeat.setCustomValidity(lang.mustMatch)
			} else {
				repeat.setCustomValidity("")
			}
		}
	}

	// Render board creation and management controls
	renderControls() {
		this.el.innerHTML = HTML
			`<a id="logout">
				${lang.logout}
			</a>`
	}

	// Log out of the user account
	logout() {
		localStorage.removeItem("sessionToken")
		localStorage.removeItem("loginID")
		location.reload()
	}
}

defer(() => panel = new AccountPanel())

// Extract login ID and password from form
function sendRequest(el: HTMLFormElement, type: message) {
	const req: any = {}
	for (let key of ['id', 'password']) {
		req[key] = (el
			.querySelector(`input[name=${key}]`) as HTMLInputElement)
			.value
	}
	loginID = req.id
	send(type, req)
}

// Both registration and login requests reply with the same messsage type
register(message.login, ({code, session}: LoginResponse) => {
	let text: string
	switch (code) {
	case responseCode.success:
		sessionToken = session
		localStorage.setItem("sessionToken", session)
		localStorage.setItem("loginID", loginID)
		write(() => panel.renderControls())
		return
	case responseCode.nameTaken:
		text = lang.nameTaken
		break
	case responseCode.wrongCredentials:
		text = lang.wrongCredentials
		break
	default:
		// These response codes are never supposed to make it here, because of
		// HTML5 form validation
		text = lang.theFuck
	}

	document.querySelector("#login-response").textContent = text
})

// Send the authentication request to the server
export function authenticate() {
	if (!sessionToken) {
		return
	}
	send(message.authenticate, {
		id: loginID,
		session: sessionToken,
	})
}

register(message.authenticate, (success: boolean) => {
	if (success) {
		write(() => panel.renderControls())
	}
})
