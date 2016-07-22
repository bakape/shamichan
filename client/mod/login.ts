// Login/logout facilities for the account system

import {TabbedModal} from '../banner'
import {write} from '../render'
import {defer} from '../defer'
import {mod as lang, ui} from '../lang'
import {
	on, loadModule, setLabel, inputValue, extend, makeEl, each
} from '../util'
import {handlers, send, message} from '../connection'
import Model from '../model'
import CaptchaView, {Captcha} from '../captcha'
import {config} from '../state'

// Login/Registration request sent to the server through websocket
interface LoginRequest extends Captcha {
	id: string
	password: string
}

// Response codes for loging in, registration and password changing
export const enum responseCode {
	success,
	nameTaken,
	wrongCredentials,
	idTooShort,
	idTooLong,
	passwordTooShort,
	passwordTooLong,
	invalidCaptcha,
}

// Login/Registration response received from the server
type LoginResponse = {
	code: responseCode
	session: string // Session ID token
}

let loginID = localStorage.getItem("loginID"),
	sessionToken = localStorage.getItem("sessionToken")

// Account login and registration
export default class AccountPanel extends TabbedModal<Model> {
	$login: HTMLFormElement = (this.el
		.querySelector("#login-form") as HTMLFormElement)
	$register: HTMLFormElement = (this.el
		.querySelector("#registration-form") as HTMLFormElement)
	captchas: {[key: string]: CaptchaView}

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

			// TODO: Log out all devices
			"#logoutAll": () =>
				alert("TODO"),

			"#changePassword":  this.loadConditionalView("mod/changePassword"),
			"#configureServer": this.loadConditionalView("mod/admin"),
			"#createBoard": this.loadConditionalView("mod/createBoard"),

			// TODO: Board configuration UI
			"#configureBoard": () =>
				alert("TODO"),
		})

		handlers[message.login] = (msg: LoginResponse) =>
			this.loginResponse(msg)
		handlers[message.authenticate]  = (success: boolean) =>
			success && this.renderControls()
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

		if (config.captcha) {
			this.captchas = {
				login: new CaptchaView("login-captcha"),
				register: new CaptchaView("registration-captcha"),
			}
		}
	}

	// Extract login ID and password from form and captcha data, if enabled
	sendRequest(el: HTMLFormElement, type: message) {
		const req: any = {}
		for (let key of ['id', 'password']) {
			req[key] = inputValue(el, key)
		}
		loginID = req.id
		if (this.captchas) {
			const captcha = this.captchas[
				type === message.login ? "login" : "register"
			]
			extend(req, captcha.data())
		}
		send(type, req)
	}

	// Handle login form
	login(event: Event) {
		event.preventDefault()
		this.sendRequest(event.target as HTMLFormElement, message.login)
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
			this.renderControls()
			return
		case responseCode.nameTaken:
			text = lang.nameTaken
			break
		case responseCode.wrongCredentials:
			text = lang.wrongCredentials
			break
		case responseCode.invalidCaptcha:
			text = ui.invalidCaptcha
			break
		default:
			// These response codes are never supposed to make it here, because
			// of HTML5 form validation
			text = lang.theFuck
		}

		if (this.captchas) {
			for (let key in this.captchas) {
				this.captchas[key].reload()
			}
		}
		renderFormResponse(this.el, text)
	}

	// Handle registration form
	register(event: Event) {
		event.preventDefault()
		this.sendRequest(event.target as HTMLFormElement, message.register)
	}

	// Render board creation and management controls
	renderControls() {
		let menu = ""
		const ids = [
			"logout", "logoutAll", "changePassword", "createBoard",
			"configureBoard"
		]
		for (let id of ids) {
			menu += this.renderLink(id)
		}
		if (loginID === "admin") {
			menu += this.renderLink("configureServer")
		}
		write(() => {
			each(this.el.children, el =>
				el.style.display = "none")
			this.el.append(makeEl(`<div class="menu">${menu}</div>`))
		})
	}

	renderLink(name: string): string {
		return `<a id="${name}">${lang[name]}</a><br>`
	}

	// Log out of the user account
	logout() {
		localStorage.removeItem("sessionToken")
		localStorage.removeItem("loginID")
		location.reload()
	}

	// Create handler for ynamically loading and rendering conditional view
	// modules
	loadConditionalView(path: string): EventListener {
		return () =>
			loadModule(path).then(m => {
				this.hideMenu()
				new m.default(this)
			})
	}

	hideMenu() {
		write(() =>
			this.el
			.querySelector(".menu")
			.style.display = "none")
	}

	unhideMenu() {
		write(() =>
			this.el
			.querySelector(".menu")
			.style.display = "")
	}
}

defer(() =>
	new AccountPanel())

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
		el2.setCustomValidity(el2.value !== el1.value ? lang.mustMatch : "")
}

// Find an input element by name within a parent form element
const findInputEl = (parent: Element, name: string) =>
	parent.querySelector(`input[name=${name}]`) as HTMLInputElement

// Render a text comment about the response status below the text
export const renderFormResponse = (el: Element, text: string) =>
	write(() =>
		el
		.querySelector(".form-response")
		.textContent = text)
