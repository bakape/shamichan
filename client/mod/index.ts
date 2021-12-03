// Login/logout/registration facilities for the account system

import { postJSON } from '../util'
import { FormView } from "../ui"
import { TabbedModal } from "../base"
import { validatePasswordMatch } from "./common"
import ModPanel from "./panel"
import { ModerationLevel } from "../common";
import {
	PasswordChangeForm, ServerConfigForm, BoardConfigForm, BoardCreationForm,
	BoardDeletionForm, StaffAssignmentForm, FormDataForm,
} from "./forms"

export { loginID, sessionToken } from "./common"


// Current staff position on this page
export const position: ModerationLevel = (window as any).position

// Only active AccountPanel instance
export let accountPanel: AccountPanel

let registrationForm: LoginForm

// Account login and registration
class AccountPanel extends TabbedModal {
	constructor() {
		super(document.getElementById("account-panel"))

		this.onClick({
			'#logout': () =>
				logout("/api/logout"),
			"#logoutAll": () =>
				logout("/api/logout-all"),
			"#changePassword": this.loadConditional(() =>
				new PasswordChangeForm()),
			"#configureServer": this.loadConditional(() =>
				new ServerConfigForm()),
			"#createBoard": this.loadConditional(() =>
				new BoardCreationForm()),
			"#deleteBoard": this.loadConditional(() =>
				new BoardDeletionForm()),
			"#configureBoard": this.loadConditional(() =>
				new BoardConfigForm()),
			"#assignStaff": this.loadConditional(() =>
				new StaffAssignmentForm()),
			"#setBanners": this.loadConditional(() =>
				new FormDataForm("/html/set-banners", "/api/set-banners")),
			"#setLoading": this.loadConditional(() =>
				new FormDataForm("/html/set-loading", "/api/set-loading")),
		})

		if (position > ModerationLevel.notStaff) {
			new ModPanel()
		}
	}

	// Create handler for dynamically loading and rendering conditional view
	// modules
	private loadConditional(module: () => void): EventListener {
		return () => {
			this.toggleMenu(false)
			module()
		}
	}

	// Either hide or show the selection menu
	public toggleMenu(show: boolean) {
		document.getElementById("form-selection")
			.style
			.display = show ? "block" : "none"
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
			location.reload()
			break
		default:
			alert(await res.text())
	}
}

// Common functionality of login and registration forms
class LoginForm extends FormView {
	private url: string

	constructor(id: string, url: string) {
		super({
			el: document.getElementById(id),
			needCaptcha: true,
		})
		this.url = "/api/" + url
	}

	// Extract and send login ID and password and captcha (if any) from a form
	protected async send() {
		const req: any = {}
		for (let key of ['id', 'password']) {
			req[key] = this.inputElement(key).value
		}

		const res = await postJSON(this.url, req)
		switch (res.status) {
			case 200:
				location.reload()
			default:
				this.renderFormResponse(await res.text())
		}
	}
}

// Init module
export default () => {
	accountPanel = new AccountPanel()
	if (position === ModerationLevel.notLoggedIn) {
		new LoginForm("login-form", "login")
		registrationForm = new LoginForm("registration-form", "register")
		validatePasswordMatch(registrationForm.el, "password", "repeat")
	}
}
