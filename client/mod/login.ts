// Login/logout facilities for the account system

import { TabbedModal } from '../banner'
import { write } from '../render'
import { loadModule } from '../util'
import { handlers, send, message } from '../connection'
import { defer } from "../defer"

// // Login/Registration response received from the server
// type LoginResponse = {
// 	code: responseCode
// 	session: string // Session ID token
// }

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
			"#configureServer": this.loadConditionalView("mod/admin"),
			"#createBoard": this.loadConditionalView("mod/createBoard"),
			"#configureBoard": this.loadConditionalView("mod/configureBoard"),
		})

		handlers[message.authenticate] = (success: boolean) => {
			if (!success) {
				localStorage.removeItem("sessionToken")
				sessionToken = ""
			}
			this.displayMenu()
		}
	}

	// Display the form selection menu
	displayMenu() {
		write(() => {
			document.getElementById("login-forms").style.display = "none"

			const m = document.getElementById("form-menu")
			m.style.display = "block"

			// Hide server configuration link, unless logged in as "admin"
			if (loginID !== "admin") {
				m.querySelector("a:last-of-type").style.display = "none"
			}
		})
	}

	// Log out of the user account
	logout() {
		localStorage.removeItem("sessionToken")
		localStorage.removeItem("loginID")
		location.reload()
	}

	// Create handler for dynamically loading and rendering conditional view
	// modules
	loadConditionalView(path: string): EventListener {
		return () =>
			loadModule(path).then(m => {
				this.toggleMenu(false)
				new m.default()
			})
	}

	// Either hide or show the selection menu
	toggleMenu(show: boolean) {
		const display = show ? "" : "none"
		write(() =>
			this.el.querySelector(".menu").style.display = display)
	}
}

defer(() =>
	new AccountPanel())

// // Common functionality of LoginForm and RegistrationForm
// class BaseLoginForm extends FormView {
// 	constructor(handler: () => void) {
// 		super({ noCancel: true }, handler)
// 	}

// 	// Extract and send login ID and password and captcha (if any) from a form
// 	sendRequest(type: message) {
// 		const req: any = {}
// 		for (let key of ['id', 'password']) {
// 			req[key] = inputValue(this.el, key)
// 		}
// 		this.injectCaptcha(req)
// 		loginID = req.id
// 		send(type, req)
// 	}

// 	// Handle the login request response from the server.
// 	// Both registration and login requests reply with the same message type
// 	loginResponse({code, session}: LoginResponse) {
// 		let text: string
// 		switch (code) {
// 			case responseCode.success:
// 				sessionToken = session
// 				localStorage.setItem("sessionToken", session)
// 				localStorage.setItem("loginID", loginID)
// 				accountPanel.displayMenu()
// 				return
// 			case responseCode.nameTaken:
// 				text = lang.nameTaken
// 				break
// 			case responseCode.wrongCredentials:
// 				text = lang.wrongCredentials
// 				break
// 			case responseCode.invalidCaptcha:
// 				text = ui.invalidCaptcha
// 				break
// 			default:
// 				// These response codes are never supposed to make it here, because
// 				// of HTML5 form validation
// 				text = lang.theFuck
// 		}

// 		this.reloadCaptcha(code)
// 		this.renderFormResponse(text)
// 	}
// }

// // Form for logging into to an existing account
// class LoginForm extends BaseLoginForm {
// 	constructor() {
// 		super(() =>
// 			this.sendRequest(message.login))
// 		this.renderForm(makeFrag(renderFields("id", "password")))

// 		handlers[message.login] = (msg: LoginResponse) =>
// 			this.loginResponse(msg)
// 	}
// }

// // Form for registering a new user account
// class RegistrationForm extends BaseLoginForm {
// 	constructor() {
// 		super(() =>
// 			this.sendRequest(message.register))
// 		this.renderForm(makeFrag(renderFields("id", "password", "repeat")))
// 		read(() =>
// 			validatePasswordMatch(this.el, "password", "repeat"))

// 		handlers[message.register] = (msg: LoginResponse) =>
// 			this.loginResponse(msg)
// 	}
// }

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
