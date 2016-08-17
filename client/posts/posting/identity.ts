// Name, email, tripcode and staff title persistence and postform propagation

import {emitChanges, ChangeEmitter} from '../../model'
import {defer} from '../../defer'
import {BannerModal} from '../../banner'
import {identity as lang} from '../../lang'
import {table, randomID} from '../../util'
import {inputType, renderInput} from '../../forms'

interface Identity extends ChangeEmitter {
	name: string
	email: string
	postPassword: string
	[index: string]: any
}

// Maximum lengths of input fields
const maxLengths: {[key: string]: number} = {
	name: 50,
	email: 100,
	auth: 50,
	postPassword: 50
}

// Values of the name and tripcode fields
const identity = {} as Identity

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

export default emitChanges(identity)

// Name and email input pannel
class IdentityPanel extends BannerModal {
	constructor() {
		super({id: "identity"})
		this.on("input", e =>
			this.onInput(e))
	}

	render() {
		const html = table(["name", "email", "postPassword"], name => {
			const [label, tooltip] = lang[name]
			return renderInput({
				name,
				label,
				tooltip,
				type: inputType.string,
				value: identity[name],
				maxLength: maxLengths[name],
			})
		})

		this.lazyRender(html)
	}

	onInput(event: Event) {
		const el = event.target as HTMLInputElement,
			name = el.getAttribute("name"),
			val = el.value
		localStorage.setItem(name, val)
		identity[name] = val
	}
}

defer(() =>
	new IdentityPanel())
