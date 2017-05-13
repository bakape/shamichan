// Login/logout/registration facilities for the account system

import { postJSON } from '../util'
import { FormView } from "../ui"
import { TabbedModal } from "../base"
import {
	validatePasswordMatch, loginID, sessionToken, deleteCookie, isAdmin
} from "./common"
import ModPanel from "./panel"
import {
	PasswordChangeForm, ServerConfigForm, BoardConfigForm, BoardCreationForm,
	BoardDeletionForm, StaffAssignmentForm,
} from "./forms"
import { config } from "../state"

export { loginID, sessionToken } from "./common"

interface Constructable {
	new (): any
}

// Only active AccountPanel instance
export let accountPanel: AccountPanel

let loginForm: LoginForm,
	registrationForm: LoginForm,
	modPanel: ModPanel

// Account login and registration
class AccountPanel extends TabbedModal {
	constructor() {
		super(document.getElementById("account-panel"))

		this.onClick({
			'#logout': () =>
				logout("/admin/logout"),
			"#logoutAll": () =>
				logout("/admin/logoutAll"),
			"#changePassword": this.loadConditional(PasswordChangeForm),
			"#configureServer": this.loadConditional(ServerConfigForm),
			"#createBoard": this.loadConditional(BoardCreationForm),
			"#deleteBoard": this.loadConditional(BoardDeletionForm),
			"#configureBoard": this.loadConditional(BoardConfigForm),
			"#assignStaff": this.loadConditional(StaffAssignmentForm),
		})

		if (loginID() && sessionToken()) {
			this.displayMenu()
		}

		this.tabHook = id => {
			switch (id) {
				case 0:
					loginForm.initCaptcha()
					break
				case 1:
					registrationForm.initCaptcha()
					break
			}
		}
		this.showHook = () => {
			if (!loginID()) {
				loginForm.initCaptcha()
			}
		}
	}

	// Display the form selection menu
	public displayMenu() {
		document.getElementById("login-forms").style.display = "none"

		const el = document.getElementById("form-selection")
		el.style.display = "block"

		// Hide some controls for non-admin accounts
		const admin = isAdmin();
		(el.lastElementChild as HTMLElement).hidden = !admin
		document.getElementById("createBoard").hidden = !admin
			&& config.disableUserBoards

		// Load Moderation panel
		modPanel = new ModPanel()
	}

	// Create handler for dynamically loading and rendering conditional view
	// modules
	private loadConditional(m: Constructable): EventListener {
		return () => {
			this.toggleMenu(false)
			new m()
		}
	}

	// Either hide or show the selection menu
	public toggleMenu(show: boolean) {
		document.getElementById("form-selection")
			.style
			.display = show ? "block" : "none"
	}
}

// Reset the views and module to its not-logged-id state
export function reset() {
	deleteCookie("loginID")
	deleteCookie("session")
	loginForm.reloadCaptcha()
	registrationForm.reloadCaptcha()
	modPanel.reset()
	document.getElementById("login-forms").style.display = ""
	document.getElementById("form-selection").style.display = "none"
	for (let el of accountPanel.el.querySelectorAll(".form-response")) {
		el.textContent = ""
	}
}

// Terminate the user session(s) server-side and reset the panel
async function logout(url: string) {
	const res = await fetch(url, {
		method: "POST",
		credentials: "include",
	})
	switch (res.status) {
		case 200:
		case 403: // Does not really matter, if the session already expired
			reset()
			break
		default:
			throw await res.text()
	}
}

// Common functionality of login and registration forms
class LoginForm extends FormView {
	private url: string

	constructor(id: string, url: string) {
		super({
			el: document.getElementById(id),
			lazyCaptcha: true,
		})
		this.url = "/admin/" + url
	}

	// Extract and send login ID and password and captcha (if any) from a form
	protected async send() {
		const req: any = {}
		for (let key of ['id', 'password']) {
			req[key] = this.inputElement(key).value
		}
		this.injectCaptcha(req)

		const res = await postJSON(this.url, req)
		switch (res.status) {
			case 200:
				accountPanel.displayMenu()

				// Clear all password fields for security reasons
				const els = this.el.querySelectorAll("input[type=password]")
				for (let el of els as HTMLInputElement[]) {
					el.value = ""
				}

				break
			default:
				this.renderFormResponse(await res.text())
		}
	}
}

// Init module
export default function () {
	accountPanel = new AccountPanel()
	loginForm = new LoginForm("login-form", "login")
	registrationForm = new LoginForm("registration-form", "register")
	validatePasswordMatch(registrationForm.el, "password", "repeat")
}
