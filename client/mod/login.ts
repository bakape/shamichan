// Login/logout facilities for the account system

import {TabbedModal} from '../banner'
import {write, read} from '../render'
import {defer} from '../defer'
import {mod as lang} from '../lang'
import {setLabel, on, HTML, loadModule, inputValue} from '../util'
import {handlers, send, message} from '../connection'

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
	sessionToken = localStorage.getItem("sessionToken")

// Account login and registration
class AccountPanel extends TabbedModal {
	$login: HTMLFormElement = (this.el
		.querySelector("#login-form") as HTMLFormElement)
	$register: HTMLFormElement = (this.el
		.querySelector("#registration-form") as HTMLFormElement)

	constructor() {
		super({el: document.querySelector('#account-panel')})

		on(this.$register, 'submit', e =>
			this.register(e))
		on(this.$login, 'submit', e =>
			this.login(e))

		validatePasswordMatch(this.$register, "password", "repeat")

		write(() =>
			this.renderInitial())

		this.onClick({
			'#logout': () =>
				this.logout(),
			// Dynamically load password change view module
			"#changePassword": e =>
				loadModule("mod/changePassword").then(m =>
					new m.default(e.target as Element))
		})

		handlers[message.login] = (msg: LoginResponse) =>
			this.loginResponse(msg)
		handlers[message.authenticate]  = (msg: boolean) =>
			this.authenticationResponse(msg)
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

	// Handle the login request response from the server.
	// Both registration and login requests reply with the same messsage type
	loginResponse({code, session}: LoginResponse) {
		let text: string
		switch (code) {
		case responseCode.success:
			sessionToken = session
			localStorage.setItem("sessionToken", session)
			localStorage.setItem("loginID", loginID)
			write(() =>
				this.renderControls())
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

		this.el
			.querySelector(".form-response")
			.textContent = text
	}

	// Handle registration form
	register(event: Event) {
		event.preventDefault()
		sendRequest(event.target as HTMLFormElement, message.register)
	}

	// Render board creation and management controls
	renderControls() {
		this.el.innerHTML = HTML
			`${this.renderLink("logout")}
			<br>
			${this.renderLink("changePassword")}
			<br>`
	}

	renderLink(name: string): string {
		return `<a id="${name}">${lang[name]}</a>`
	}

	// Log out of the user account
	logout() {
		localStorage.removeItem("sessionToken")
		localStorage.removeItem("loginID")
		location.reload()
	}

	// Handle authentication response message
	authenticationResponse(success: boolean) {
		success && write(() =>
			this.renderControls())
	}
}

defer(() =>
	new AccountPanel())

// Extract login ID and password from form
function sendRequest(el: HTMLFormElement, type: message) {
	const req: any = {}
	for (let key of ['id', 'password']) {
		req[key] = inputValue(el, key)
	}
	loginID = req.id
	send(type, req)
}

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

// Set a password match validator function for 2 input elements, that are
// children of the passed element.
export function validatePasswordMatch (
	parent: Element, name1: string, name2: string
) {
	const el1 = findInputEl(parent, name1),
		el2 = findInputEl(parent, name2)
	el2.onchange = () =>
		el2.value !== el1.value
			? el2.setCustomValidity(lang.mustMatch)
			: el2.setCustomValidity("")
}

const findInputEl = (parent: Element, name: string) =>
	parent.querySelector(`input[name=${name}]`) as HTMLInputElement
