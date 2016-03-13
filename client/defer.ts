/*
 Seperate module for defering operations to execute. Helps avoid race
 conditions, circular dependancies and offload less critical expensive
 computations to later.
*/

type DeferedFunc = () => void

const deferred: DeferedFunc[] = []

// Add a function to be executed, once the module finishes loading
export function defer(func: DeferedFunc) {
	deferred.push(func)
}

// Execute all stored deferred functions
export function exec() {
	while (deferred.length > 0) {
		deferred.shift()()
	}
}
