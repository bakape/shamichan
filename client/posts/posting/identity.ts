// Name, email, tripcode and staff title persistence and postform propagation

import { emitChanges, ChangeEmitter } from '../../model'
import { randomID } from '../../util'
import { BannerModal } from "../../banner"
import { write } from "../../render"

interface Identity extends ChangeEmitter {
	name: string
	email: string
	postPassword: string
	[index: string]: any
}

// Base of any post allocation request
export interface PostCredentials {
	name?: string
	email?: string
	auth?: string // TODO
	password?: string
	[index: string]: any
}

// Values of the name and tripcode fields
const identity = emitChanges({} as Identity)
export default identity

// Load from localStorage or initialize
for (let name of ["name", "email"]) {
	identity[name] = localStorage.getItem(name) || ""
}
let stored = localStorage.getItem("postPassword")
if (!stored) {
	stored = randomID(32)
	localStorage.setItem("postPassword", stored)
}
identity.postPassword = stored

// Name and email input panel
class IdentityPanel extends BannerModal {
	constructor() {
		super(document.getElementById("identity"))
		this.on("input", e =>
			this.onInput(e))
		this.assignValues()
	}

	assignValues() {
		write(() => {
			for (let key of ["name", "email", "postPassword"]) {
				(this.el.querySelector(`input[name=${key}]`) as HTMLInputElement)
					.value = identity[key]
			}
		})
	}

	onInput(event: Event) {
		const el = event.target as HTMLInputElement,
			name = el.getAttribute("name"),
			val = el.value
		localStorage.setItem(name, val)
		identity[name] = val
	}
}

new IdentityPanel()

// Generate a new base post allocation request
export function newAllocRequest(): PostCredentials {
	const req: PostCredentials = {
		password: identity.postPassword,
	} as any

	for (let key of ["name", "email"]) {
		const val = identity[key]
		if (val) {
			req[key] = val
		}
	}

	return req
}
