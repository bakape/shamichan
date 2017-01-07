// Name, tripcode and staff title persistence and postform propagation

import { BannerModal } from '../../base'
import { extend, write, emitChanges, ChangeEmitter } from '../../util'
import { newRequest } from "../../mod"


const base64 =
	'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_'
		.split("")
const authCheckbox = document.getElementById("staffTitle") as HTMLInputElement

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
	identity.postPassword = randomID(32)
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
		write(() => {
			for (let key of ["name", "postPassword"]) {
				(this.el.querySelector(`input[name=${key}]`) as HTMLInputElement)
					.value = identity[key]
			}
		})
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

// Generate a random base64 string of passed length
function randomID(len: number): string {
	let id = ''
	for (let i = 0; i < len; i++) {
		id += random(base64)
	}
	return id
}

// Return a random item from an array
function random<T>(array: T[]): T {
	return array[Math.floor(Math.random() * array.length)]
}

export function initIdentity() {
	new IdentityPanel()
}
