import View from '../view'
import {write} from '../render'
import {handlers, send, message} from '../connection'
import {InputSpec, renderInput, inputType, FormView} from './util'
import {admin as lang, fetchAdminPack, mod} from '../lang'
import AccountPanel from './login'
import {HTML} from '../util'

type ServerConfigs = {
	prune: boolean
	radio: boolean
	hats: boolean
	illyaDance: boolean
	pyu: boolean
	maxWidth: number
	maxHeight: number
	maxThreads: number
	maxBump: number
	JPEGQuality: number
	PNGQuality: number
	threadCooldown: number
	maxSubjectLength: number
	maxSize: number
	sessionExpiry: number
	defaultLang: string
	frontPage: string
	origin: string
	defaultCSS: string
	salt: string
	excludeRegex: string
	feedbackEmail: string
	FAQ: string
	langs: string[]
	links: string[][]
	spoilers: number[]
	[index: string]: any
}

const specs: InputSpec[] = [
	{
		name: "prune",
		type: inputType.boolean,
	},
	{
		name: "radio",
		type: inputType.boolean,
	},
	{
		name: "pyu",
		type: inputType.boolean,
	},
	{
		name: "illyaDance",
		type: inputType.boolean,
	},
	{
		name: "hats",
		type: inputType.boolean,
	},
]

// Panel for server administration controls such as global server settings
export default class ConfigPanel extends FormView {
	constructor(parent: AccountPanel) {
		super(parent, el =>
			this.extractConfigs(el))

		// Request curent configuration and render the panel
		send(message.configServer, null)
		handlers[message.configServer] = async (conf: ServerConfigs) => {
			delete handlers[message.configServer]
			await fetchAdminPack()
			this.render(conf)
		}
	}

	// Render the panel element contents
	render(conf: ServerConfigs) {
		let inputs = ""
		for (let spec of specs) {
			const ln = lang[spec.name]
			spec.label = ln[0]
			spec.tooltip = ln[1]
			spec.value = conf[spec.name]
			inputs += renderInput(spec)
		}
		this.renderForm(inputs)
	}

	// Clean up any dangling references and GC the view
	remove() {
		delete handlers[message.configServer]
		super.remove()
	}

	// Exteract the configuration struct from the form
	extractConfigs(form: Element) {
		const conf: {[key: string]: any} = {}
		const els = form
			.querySelectorAll("input") as NodeListOf<HTMLInputElement>
		for (let el of els) {
			let val: any
			switch (el.type) {
			case "submit":
			case "button":
				continue
			case "checkbox":
				val = el.checked
			}
			conf[el.name] = val
		}
		console.log(conf)
	}
}
