// Login/logout/registration facilities for the account system

import { TabbedModal } from '../banner'
import { write } from '../render'
import { loadModule, inputValue } from '../util'
import FormView from "../forms"
import { validatePasswordMatch } from "./common"
import { postJSON } from "../fetch"

// User ID and session ID currently in use
export let loginID = localStorage.getItem("loginID"),
	sessionToken = localStorage.getItem("sessionToken"),
	// Only active AccountPanel instance
	accountPanel: AccountPanel

// Account login and registration
class AccountPanel extends TabbedModal {
	constructor() {
		super(document.getElementById("account-panel"))
		accountPanel = this

		this.onClick({
			'#logout': reset,

			// TODO: Log out all devices
			"#logoutAll": () =>
				alert("TODO"),

			"#changePassword": this.loadConditionalView("mod/changePassword"),
			"#configureServer": this.loadConditionalView("mod/configureServer"),
			"#createBoard": this.loadConditionalView("mod/createBoard"),
			"#configureBoard": this.loadConditionalView("mod/configureBoard"),
		})

		if (loginID && sessionToken) {
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

// Reset the views and module to its not-logged-id state
export function reset() {
	localStorage.removeItem("sessionToken")
	localStorage.removeItem("loginID")
	loginID = ""
	sessionToken = ""
	write(() => {
		document.getElementById("login-forms").style.display = ""
		document.getElementById("form-selection").style.display = "none"
		for (let el of accountPanel.el.querySelectorAll(".form-response")) {
			el.textContent = ""
		}
	})
}

// Common functionality of login and registration forms
class BaseLoginForm extends FormView {
	private url: string

	constructor(id: string, url: string) {
		super({ el: document.getElementById(id) }, () =>
			this.sendRequest())
		this.url = "/admin/" + url
	}

	// Extract and send login ID and password and captcha (if any) from a form
	private async sendRequest() {
		const req: any = {}
		for (let key of ['id', 'password']) {
			req[key] = inputValue(this.el, key)
		}
		this.injectCaptcha(req)
		loginID = req.id

		const res = await postJSON(this.url, req)
		switch (res.status) {
			case 200:
				const token = await res.text()
				sessionToken = token
				localStorage.setItem("sessionToken", token)
				localStorage.setItem("loginID", loginID)
				accountPanel.displayMenu()

				// Clear all password fields for security reasons
				write(() => {
					const els = this.el.querySelectorAll("input[type=password]")
					for (let el of els as HTMLInputElement[]) {
						el.value = ""
					}
				})

				break
			default:
				this.renderFormResponse(await res.text())
				this.reloadCaptcha()
		}
	}
}

// Init module
new AccountPanel()
new BaseLoginForm("login-form", "login")
validatePasswordMatch(
	new BaseLoginForm("registration-form", "register").el,
	"password",
	"repeat",
)

