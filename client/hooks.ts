// Hooks for optional modules to execute code in exposed functions

type Hook = (arg?: any) => any
type HookMap = {[key: string]: Hook[]}

// Hooks for optional handlers
export const hooks: HookMap = {}

// Assigns a handler to execute on a hook name
export function hook(name: string, func: Hook) {
	const hook = hooks[name]
	if (!hook) {
		hooks[name] = [func]
	} else {
		hook.push(func)
	}
}

// Execute all handlers for a hook
export function trigger(name: string, param?: any) {
	const hook = hooks[name]
	if (!hook) {
		return
	}
	for (let func of hook) {
		func(param)
	}
}
