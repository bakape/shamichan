// Name, tripcode and staff title persistence and postform propagation

import { BannerModal } from '../../base'
import { extend, emitChanges, ChangeEmitter } from '../../util'
import { newRequest } from "../../mod"

const base64 = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_',
	authCheckbox = document.getElementById("staffTitle") as HTMLInputElement

interface Identity extends ChangeEmitter {
	auth: boolean
	name: string
	postPassword: string
	[index: string]: any
}

let identity = {
	auth: false,
	name: localStorage.getItem("name") || "",
	postPassword: localStorage.getItem("postPassword") || "",
} as Identity
if (!identity.postPassword) {
	identity.postPassword = randomID(64)
	localStorage.setItem("postPassword", identity.postPassword)
}
export default identity = emitChanges(identity)

// Poster identity input panel
class IdentityPanel extends BannerModal {
	constructor() {
		super(document.getElementById("identity"))
		this.on("input", e =>
			this.onInput(e))
		authCheckbox.addEventListener("change", () => this.onAuthChange(), {
			passive: true,
		})
		this.assignValues()
	}

	private assignValues() {
		(this.el.querySelector(`input[name="name"]`) as HTMLInputElement)
			.value = identity.name
	}

	private onInput(event: Event) {
		const el = event.target as HTMLInputElement,
			name = el.getAttribute("name"),
			val = el.value
		localStorage.setItem(name, val)
		identity[name] = val
	}

	private onAuthChange() {
		identity.auth = authCheckbox.checked
	}
}

// Generate a new base post allocation request
export function newAllocRequest() {
	const req = { password: identity.postPassword } as any
	if (identity.name) {
		req["name"] = identity.name
	}
	if (authCheckbox.checked) {
		extend(req, newRequest())
	}
	return req
}

// Generate a random unpadded base64 string of passed byte length
function randomID(len: number): string {
	let id = ''
	const b = new Uint8Array(len)
	crypto.getRandomValues(b)
	for (let i = 0; i < len; i++) {
		id += base64[b[i] % 64]
	}
	return id
}

export function initIdentity() {
	new IdentityPanel()
}
