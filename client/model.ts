import {extend} from 'underscore'
import Collection from './collection'

type ModelAttrs = {[attr: string]: any}
type HookHandler = (arg: any) => void
type HookMap = {[key: string]: HookHandler[]}

// Generic model class, that all other model classes extend
export default class Model {
	attrs: ModelAttrs
	id: string|number
	collection: Collection
	private changeHooks: HookMap = {}

	constructor(attrs: ModelAttrs = {}) {
		this.attrs = attrs
	}

	// Retrieve a stored value of specific key from the model's attribute
	// object
	get(key: string): any {
		return this.attrs[key]
	}

	// Set a key to a target value
	set(key: string, val: any) {
		this.attrs[key] = val
		this.execChangeHooks(key, val)
	}

	// Extend the model attribute hash, with the suplied object. Shorthand, for
	// setting multiple fields simultaniously.
	setAttrs(attrs: ModelAttrs) {
		extend(this.attrs, attrs)
		for (let key in attrs) {
			this.execChangeHooks(key, attrs[key])
		}
	}

	// Append value to an array strored at the given key. If the array does not
	// exist, it is created.
	append(key: string, val: any) {
		if (this.attrs[key]) {
			this.attrs[key].push(val)
		} else {
			this.attrs[key] = [val]
		}
		this.execChangeHooks(key, this.get(key))
	}

	// Extend an object at target key. If key does not exist, simply assign the
	// object to the key.
	extend(key: string, object: Object) {
		if (this.attrs[key]) {
			extend(this.attrs[key], object)
		} else {
			this.attrs[key] = object
		}
		this.execChangeHooks(key, this.get(key))
	}

	// Add a function to be executed, when .set(), .setAttrs(), .append() or
	// .extend() modify a key's value.
	onChange(key: string, func: HookHandler) {
		if (this.changeHooks[key]) {
			this.changeHooks[key].push(func)
		} else {
			this.changeHooks[key] = [func]
		}
	}

	// Execute handlers hooked into key change, if any
	private execChangeHooks(key: string, val: any) {
		const hooks = this.changeHooks[key]
		if (!hooks) {
			return
		}
		for (let func of hooks) {
			func(val)
		}
	}

	// Remove the model from its collection, detach all references and allow to
	// be garbage collected.
	remove() {
		if (this.collection) {
			this.collection.remove(this)
		}
		delete this.changeHooks
		delete this.attrs
	}
}
