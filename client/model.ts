import Collection from './collection'
import View from './view'

export type ModelAttrs = {[attr: string]: any}

// Generic model class, that all other model classes extend
export default class Model {
	id: number
	collection: Collection<Model>
	views: Set<View<Model>> = new Set<View<Model>>()

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
	attach(view: View<Model>) {
		this.views.add(view)
	}

	// Detach a view from the model
	detach(view: View<Model>) {
		this.views.delete(view)
	}

	[index: string]: any
}

// Wrap an object with a Proxy that executes handlers on property changes.
// To add new handlers, call the .onChange method on the object.
// For type safety, the passed generic interface must extend ChangeEmitter.
export function emitChanges<T extends ChangeEmitter>(obj: T = {} as T): T {
	const changeHooks: HookMap = {}

	const proxy = new Proxy<T>(obj, {
		set(target: T, key: string, val: any) {
			(target as any)[key] = val

			// Execute handlers hooked into the key change, if any
			const hooks = changeHooks[key]
			if (hooks) {
				for (let func of hooks) {
					func(val)
				}
			}

			return true
		},
	})

	// Add a function to be executed, when a key is set on the object.
	// Proxies do not have a prototype. Some hacks required.
	proxy.onChange = (key: string, func: HookHandler) => {
		const hooks = changeHooks[key]
		if (hooks) {
			hooks.push(func)
		} else {
			changeHooks[key] = [func]
		}
	}

	return proxy
}
