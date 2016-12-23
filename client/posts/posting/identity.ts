// Name, tripcode and staff title persistence and postform propagation

import { emitChanges, ChangeEmitter } from '../../model'
import { randomID, extend } from '../../util'
import { BannerModal } from "../../banner"
import { write } from "../../render"
import { newRequest } from "../../mod/common"

interface Identity extends ChangeEmitter {
	name: string
	postPassword: string
	[index: string]: any
}

// Values of the name and tripcode fields
let identity = {} as Identity

// Load from localStorage or initialize
identity.name = localStorage.getItem("name") || ""
let stored = localStorage.getItem("postPassword")
if (!stored) {
	stored = randomID(32)
	localStorage.setItem("postPassword", stored)
}
identity.postPassword = stored
export default identity = emitChanges(identity)

// Poster identity input panel
class IdentityPanel extends BannerModal {
	constructor() {
		super(document.getElementById("identity"))
		this.on("input", e =>
			this.onInput(e))
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
}

new IdentityPanel()

// Generate a new base post allocation request
export function newAllocRequest() {
	const req = { password: identity.postPassword } as any

	if (identity.name) {
		req["name"] = identity.name
	}
	if ((document.getElementById("staffTitle") as HTMLInputElement).checked) {
		extend(req, newRequest())
	}

	return req
}
