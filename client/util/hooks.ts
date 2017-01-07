// Hooks for optional modules to execute code in exposed functions

type Hook = (...args: any[]) => any
type HookMap = { [key: string]: Hook }

// Hooks for optional handlers
const hooks: HookMap = {}

// Assigns a handler to execute on a hook name
export function hook(name: string, func: Hook) {
	hooks[name] = func
}

// Execute all handlers for a hook
export function trigger(name: string, ...args: any[]): any | null {
	const func = hooks[name]
	if (!func) {
		return undefined
	}
	return func(...args)
}
