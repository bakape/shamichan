// Login/logout facilities for the account system

import {TabbedModal} from '../banner'
import {write, read} from '../render'
import {defer} from '../defer'
import {mod as lang, ui} from '../lang'
import {loadModule, inputValue, HTML} from '../util'
import {handlers, send, message} from '../connection'
import {FormView} from '../forms'
import {renderFields, validatePasswordMatch} from './common'

// Login/Registration response received from the server
type LoginResponse = {
	code: responseCode
	session: string // Session ID token
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

let loginID = localStorage.getItem("loginID"),
	sessionToken = localStorage.getItem("sessionToken")

// Only active AccountPanel instance
export let accountPannel: AccountPanel

// Account login and registration
export default class AccountPanel extends TabbedModal {
	// Switched between this.renderInitial and this.renderControls() at runtime
	render: () => void

	constructor() {
		super({id: "account-panel"})
		accountPannel = this

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

		this.render = this.renderInitial
		handlers[message.authenticate]  = (success: boolean) => {
			if (success) {
				this.render = this.renderControls
				if (this.isRendered) {
					this.render()
				}
			}
		}
	}

	// Render the login an redistration forms in a tabbed panel
	renderInitial() {
		const html = HTML
			`<div class="tab-butts">
				<a class="tab-link tab-sel" data-id="0">
					${lang.id}
				</a>
				<a class="tab-link" data-id="1">
					${lang.register}
				</a>
			</div>
			<hr>
			<div class="tab-cont">
				<div class="tab-sel" data-id="0"></div>
				<div data-id="1"></div>
			</div>`

		this.lazyRender(html)
		read(() => {
			const tabs = this.el.querySelectorAll(".tab-cont div")
			write(() => {
				tabs[0].append(new LoginForm().el)
				tabs[1].append(new RegistrationForm().el)
			})
		})
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

		this.lazyRender(`<div class="menu">${menu}</div>`)
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
				new m.default()
			})
	}

	hideMenu() {
		write(() =>
			(this.el.querySelector(".menu") as HTMLElement)
			.style.display = "none")
	}

	unhideMenu() {
		write(() =>
			(this.el.querySelector(".menu") as HTMLElement)
			.style.display = "")
	}
}

defer(() =>
	new AccountPanel())

// Common functionality of LoginForm and RegistrationForm
class BaseLoginForm extends FormView {
	constructor(handler: () => void) {
		super({noCancel: true}, handler)
	}

	// Extract and send login ID and password and captcha (if any) from a form
	sendRequest(type: message) {
		const req: any = {}
		for (let key of ['id', 'password']) {
			req[key] = inputValue(this.el, key)
		}
		this.injectCaptcha(req)
		loginID = req.id
		send(type, req)
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
			accountPannel.renderControls()
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

		this.reloadCaptcha(code)
		this.renderFormResponse(text)
	}
}

// Form for loggin into to an existing account
class LoginForm extends BaseLoginForm {
	constructor() {
		super(() =>
			this.sendRequest(message.login))
		this.renderForm(renderFields("id", "password"))

		handlers[message.login] = (msg: LoginResponse) =>
			this.loginResponse(msg)
	}
}

// Form for registering a new user account
class RegistrationForm extends BaseLoginForm {
	constructor() {
		super(() =>
			this.sendRequest(message.register))
		this.renderForm(renderFields("id", "password", "repeat"))
		read(() =>
			validatePasswordMatch(this.el, "password", "repeat"))

		handlers[message.register] = (msg: LoginResponse) =>
			this.loginResponse(msg)
	}
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
