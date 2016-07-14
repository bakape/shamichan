export type ModelAttrs = {[attr: string]: any}

// Generic model class, that all other model classes extend
export default class Model {
	id: number

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
