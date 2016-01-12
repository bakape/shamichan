/**
 * Hooks for optional handlers
 */
export const hooks = {}

/**
 * Assigns a handler to execute on a hook name
 * @param {string} name
 * @param {function} func
 */
export function hook(name, func) {
	const hook = hooks[name]
	if (!hook) {
		hooks[name] = [func]
	} else {
		hook.push(func)
	}
}

/**
 * Execute all handlers for a hook
 * @param {string} name
 * @param {*} param
 */
export function trigger(name, param) {
	const hook = hooks[name]
	if (!hook) {
		return
	}
	for (let func of hook) {
		func(param)
	}
}
