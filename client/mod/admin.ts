import View from '../view'
import {write} from '../render'
import {handlers, send, message} from '../connection'
import {InputSpec, renderInput, inputType, FormView} from './util'
import {admin as lang, fetchAdminPack, mod} from '../lang'
import AccountPanel from './login'
import {HTML} from '../util'
import {langs, themes} from '../options/specs'

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
	origin: string
	salt: string
	excludeRegex: string
	feedbackEmail: string
	FAQ: string
	defaultCSS: string
	defaultLang: string
	links: string[][]
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
	{
		name: "maxWidth",
		type: inputType.number,
		min: 1,
	},
	{
		name: "maxHeight",
		type: inputType.number,
		min: 1,
	},
	{
		name: "maxSize",
		type: inputType.number,
		min: 1,
	},
	{
		name: "JPEGQuality",
		type: inputType.number,
		min: 1,
		max: 100,
	},
	{
		name: "PNGQuality",
		type: inputType.number,
		min: 1,
	},
	{
		name: "maxThreads",
		type: inputType.number,
		min: 1,
	},
	{
		name: "maxBump",
		type: inputType.number,
		min: 1,
	},
	{
		name: "threadCooldown",
		type: inputType.number,
		min: 0,
	},
	{
		name: "maxSubjectLength",
		type: inputType.number,
		min: 1,
	},
	{
		name: 'sessionExpiry',
		type: inputType.number,
		min: 1,
	},
	{
		name: "origin",
		type: inputType.string,
	},
	{
		name: "salt",
		type: inputType.string,
	},
	{
		name: "feedbackEmail",
		type: inputType.string,
	},
	{
		name: "defaultLang",
		type: inputType.select,
		choices: langs,
	},
	{
		name: "defaultCSS",
		type: inputType.select,
		choices: themes,
	},
	{
		name: "FAQ",
		type: inputType.multiline,
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
			.querySelectorAll("input,select,textarea") as NodeListOf<HTMLInputElement>
		for (let el of els) {
			let val: any
			switch (el.type) {
			case "submit":
			case "button":
				continue
			case "checkbox":
				val = el.checked
				break
			case "number":
				val = parseInt(el.value)
				break
			default:
				val = el.value
			}
			conf[el.name] = val
		}
		send(message.configServer, conf)
		this.remove()
	}
}
