import { handlers, send, message } from '../connection'
import {InputSpec, renderInput, inputType} from '../forms'
import AccountFormView from './common'
import {admin as lang, fetchAdminPack} from '../lang'
import {table, makeFrag} from '../util'
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
	maxSize: number
	sessionExpiry: number
	origin: string
	salt: string
	excludeRegex: string
	feedbackEmail: string
	FAQ: string
	defaultCSS: string
	defaultLang: string
	links: {[key: string]: string}

	[index: string]: any
}

const specs: InputSpec[] = [
	{
		name: "prune",
		type: inputType.boolean,
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
		name: "salt",
		type: inputType.string,
	},
	{
		name: "captcha",
		type: inputType.boolean,
	},
	{
		name: "captchaPublicKey",
		type: inputType.string,
	},
	{
		name: "captchaPrivateKey",
		type: inputType.string,
	},
	{
		name: 'sessionExpiry',
		type: inputType.number,
		min: 1,
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
		name: "radio",
		type: inputType.boolean,
	},
	{
		name: "illyaDance",
		type: inputType.boolean,
	},
	{
		name: "pyu",
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
		name: "FAQ",
		type: inputType.multiline,
	},
	{
		name: "links",
		type: inputType.map
	}
]

// Panel for server administration controls such as global server settings
export default class ConfigPanel extends AccountFormView {
	constructor() {
		const attrs = {
			class: 'wide-fields', // The panel needs much larger text inputs
			noCaptcha: true,
		}
		super(attrs, () =>
			this.extractConfigs())

		// Request curent configuration and render the panel
		send(message.configServer, null)
		handlers[message.configServer] = (conf: ServerConfigs) =>
			this.handleResponse(conf)
	}

	// Render the panel element contents
	render(conf: ServerConfigs) {
		const html = table(specs, spec =>
			([spec.label, spec.tooltip] = lang[spec.name],
			spec.value = conf[spec.name],
			renderInput(spec)))
		this.renderForm(makeFrag(html))
	}

	// Clean up any dangling references and GC the view
	remove() {
		delete handlers[message.configServer]
		super.remove()
	}

	// Extract the configuration struct from the form
	extractConfigs() {
		const req = {} as ServerConfigs
		const els = this.el
			.querySelectorAll("input[name], select[name], textarea[name]")

		for (let el of els as HTMLInputElement[]) {
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
			req[el.name] = val
		}

		// Read links key-value pairs
		const keyVals = this.el.querySelectorAll(
			"div[name=links] .map-field"
		) as HTMLInputElement[]
		req.links = {}
		for (let i = 0; i < keyVals.length; i += 2) {
			req.links[keyVals[i].value] = keyVals[i+1].value
		}

		send(message.configServer, req)
		this.remove()
	}

	// Handle server response
	async handleResponse(conf: ServerConfigs) {
		delete handlers[message.configServer]
		await fetchAdminPack()
		this.render(conf)
	}
}
