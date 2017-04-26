// Name, tripcode and staff title persistence and postform propagation

import { BannerModal } from '../../base'
import { extend, emitChanges, ChangeEmitter } from '../../util'
import { newRequest } from "../../mod"

const base64 = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_'

interface Identity extends ChangeEmitter {
	auth: boolean
	name: string
	sage: boolean
	postPassword: string
	[index: string]: any
}

const identity = emitChanges({
	auth: false,
	name: localStorage.getItem("name") || "",
	sage: localStorage.getItem("sage") === "true",
	postPassword: randomID(64),
} as Identity)
export default identity

// Poster identity input panel
class IdentityPanel extends BannerModal {
	constructor() {
		super(document.getElementById("identity"))
		this.on("input", this.onInput.bind(this), {
			passive: true,
			selector: `input[type=text]`,
		})
		this.on("change", this.onCheckboxChange.bind(this), {
			passive: true,
			selector: `input[type=checkbox]`,
		})
		this.assignValues()
	}

	private assignValues() {
		for (let el of this.el.querySelectorAll("input") as HTMLInputElement[]) {
			const name = el.getAttribute("name")
			switch (el.getAttribute("type")) {
				case "text":
					el.value = identity[name]
					break
				case "checkbox":
					el.checked = identity[name]
					break
			}
		}
	}

	private onInput(event: Event) {
		const el = event.target as HTMLInputElement,
			name = el.getAttribute("name"),
			val = el.value
		localStorage.setItem(name, val)
		identity[name] = val
	}

	private onCheckboxChange(e: Event) {
		const el = event.target as HTMLInputElement,
			name = el.getAttribute("name"),
			val = el.checked
		if (name === "staffTitle") {
			identity["auth"] = val
			return
		}
		identity[name] = val
		localStorage.setItem(name, val.toString())
	}
}

// Generate a new base post allocation request
export function newAllocRequest() {
	const req: { [key: string]: any } = { password: identity.postPassword }
	for (let key of ["name", "sage"]) {
		if (identity[key]) {
			req[key] = identity[key]
		}
	}
	if (identity.auth) {
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
