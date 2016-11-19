// Login/logout/registration facilities for the account system

import { TabbedModal } from '../banner'
import { write } from '../render'
import { loadModule, inputValue } from '../util'
import { handlers, send, message } from '../connection'
import { defer } from "../defer"
import FormView from "../forms"
import { mod as lang, ui } from "../lang"
import { validatePasswordMatch } from "./common"

// Login/Registration response received from the server
type LoginResponse = {
	code: responseCode
	session: string // Session ID token
}

// Response codes for logging in, registration and password changing
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

// User ID and session ID currently in use
export let loginID = localStorage.getItem("loginID"),
	sessionToken = localStorage.getItem("sessionToken")

// Only active AccountPanel instance
export let accountPanel: AccountPanel

// Account login and registration
export default class AccountPanel extends TabbedModal {
	constructor() {
		super(document.getElementById("account-panel"))
		accountPanel = this

		this.onClick({
			'#logout': () =>
				this.logout(),

			// TODO: Log out all devices
			"#logoutAll": () =>
				alert("TODO"),

			"#changePassword": this.loadConditionalView("mod/changePassword"),
			"#configureServer": this.loadConditionalView("mod/configureServer"),
			"#createBoard": this.loadConditionalView("mod/createBoard"),
			"#configureBoard": this.loadConditionalView("mod/configureBoard"),
		})

		new LoginForm(document.getElementById("login-form"))
		new RegistrationForm(document.getElementById("registration-form"))

		handlers[message.authenticate] = (success: boolean) => {
			if (!success) {
				localStorage.removeItem("sessionToken")
				sessionToken = ""
			}
			this.displayMenu()
		}
	}

	// Display the form selection menu
	public displayMenu() {
		write(() => {
			document.getElementById("login-forms").style.display = "none"

			const el = document.getElementById("form-selection")
			el.style.display = "block"

			// Hide server configuration link, unless logged in as "admin"
			if (loginID !== "admin") {
				(el.lastElementChild as HTMLElement).style.display = "none"
			}
		})
	}

	// Log out of the user account
	private logout() {
		localStorage.removeItem("sessionToken")
		localStorage.removeItem("loginID")
		location.reload()
	}

	// Create handler for dynamically loading and rendering conditional view
	// modules
	private loadConditionalView(path: string): EventListener {
		return () =>
			loadModule(path).then(m => {
				this.toggleMenu(false)
				new m.default()
			})
	}

	// Either hide or show the selection menu
	public toggleMenu(show: boolean) {
		const display = show ? "block" : "none"
		write(() =>
			document.getElementById("form-selection").style.display = display)
	}
}

defer(() =>
	new AccountPanel())

// Common functionality of LoginForm and RegistrationForm
class BaseLoginForm extends FormView {
	constructor(el: HTMLElement, handler: () => void) {
		super({ el }, handler)
	}

	// Extract and send login ID and password and captcha (if any) from a form
	protected sendRequest(type: message) {
		const req: any = {}
		for (let key of ['id', 'password']) {
			req[key] = inputValue(this.el, key)
		}
		this.injectCaptcha(req)
		loginID = req.id
		send(type, req)
	}

	// Handle the login request response from the server.
	// Both registration and login requests reply with the same message type
	protected loginResponse({code, session}: LoginResponse) {
		let text: string
		switch (code) {
			case responseCode.success:
				sessionToken = session
				localStorage.setItem("sessionToken", session)
				localStorage.setItem("loginID", loginID)
				accountPanel.displayMenu()
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

		this.reloadCaptcha()
		this.renderFormResponse(text)
	}
}

// Form for logging into to an existing account
class LoginForm extends BaseLoginForm {
	constructor(el: HTMLElement) {
		super(el, () =>
			this.sendRequest(message.login))
		handlers[message.login] = (msg: LoginResponse) =>
			this.loginResponse(msg)
	}
}

// Form for registering a new user account
class RegistrationForm extends BaseLoginForm {
	constructor(el: HTMLElement) {
		super(el, () =>
			this.sendRequest(message.register))
		validatePasswordMatch(this.el, "password", "repeat")
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
