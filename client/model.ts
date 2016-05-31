import {extend} from './util'
import Collection from './collection'
import View from './view'

export type ModelAttrs = {[attr: string]: any}
export type HookHandler = (arg: any) => void
export type HookMap = {[key: string]: HookHandler[]}

// Generic model class, that all other model classes extend
export default class Model {
	id: number
	collection: Collection<Model>
	views: Set<View> = new Set<View>()

	constructor() {}

	// Remove the model from its collection, detach all references and allow to
	// be garbage collected.
	remove() {
		if (this.collection) {
			this.collection.remove(this)
		}
		for (let view of this.views) {
			view.remove()
		}
	}

	// Attach a view to the model. Each model can have several views attached to
	// it.
	attach(view: View) {
		this.views.add(view)
	}

	// Detach a view from the model
	detach(view: View) {
		this.views.delete(view)
	}
}

// An extension of Model, which supports eventful hooks on attribute change
export class EventfulModel<K> extends Model {
	attrs: ModelAttrs
	changeHooks: HookMap = {}

	constructor(attrs: {[key: string]: any} = {}) {
		super()
		this.attrs = attrs
	}

	// Add a function to be executed, when .set(), .setAttrs(), .append() or
	// .extend() modify a key's value.
	onChange(key: K, func: HookHandler) {
		const hooks = this.changeHooks[key as any]
		if (hooks) {
			hooks.push(func)
		} else {
			this.changeHooks[key as any] = [func]
		}
	}

	// Execute handlers hooked into key change, if any
	execChangeHooks(key: K, val: any) {
		const hooks = this.changeHooks[key as any]
		if (!hooks) {
			return
		}
		for (let func of hooks) {
			func(val)
		}
	}

	// Retrieve a stored value of specific key from the model's attribute
	// object
	get(key: K): any {
		return this.attrs[key as any]
	}

	// Set a key to a target value
	set(key: K, val: any) {
		this.attrs[key as any] = val
		this.execChangeHooks(key , val)
	}

	// Remove the model from its collection, detach all references and allow to
	// be garbage collected.
	remove() {
		super.remove()
		delete this.attrs
		delete this.changeHooks
	}
}
