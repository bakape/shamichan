/*
 Separate module for deferring operations to execute. Helps avoid race
 conditions, circular dependencies and offload less critical expensive
 computations to later.
*/

type Func = () => void

let deferred: Func[] = [],
	inits: Func[] = []

// Add a function to be executed, once the module finishes loading
export function defer(func: Func) {
	deferred.push(func)
}

// Execute all stored deferred functions
export function exec() {
	for (let fn of deferred) {
		fn()
	}
	deferred = []
}

// Defer initialization functions to be loaded after the main infrastructure
// like "state" and "connection" modules are loaded
export function deferInit(fn: Func) {
	inits.push(fn)
}

// Execute all deferred initialization functions
export function init() {
	for (let fn of inits) {
		fn()
	}
	inits = []
}
