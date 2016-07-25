// Name, email, tripcode and staff title persistence and postform
// propagation

import {emitChanges, ChangeEmitter} from '../model'
import {write, read} from '../render'
import {defer} from '../defer'

interface Identity extends ChangeEmitter {
	name: string
	email: string
}

// Values of the name and tripcode fields
const identity: Identity = {
	name: "",
	email: "",
} as Identity
export default identity

defer(() =>
	read(() => {
		for (let id of ["name", "email"]) {
			listenToField(id)
		}
	}))


// Iniitialize, listem to and propagate changes of an identity field
function listenToField(name: string) {
	const el = document.getElementById(name)
	write(() => {
		el.value = localStorage.getItem(name)
		el.addEventListener("input", () => {
			let val = el.value.trim()
			identity[name] = val
			if (name === "email" && val === "sage") {
				val = ""
			}
			localStorage.setItem(name, val)
		})
	})
}
