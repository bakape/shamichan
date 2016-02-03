/**
 * Seperate module for defering operations to execute. Helps avoid race
 * conditions, circular dependancies and offload less critical expensive
 * computations to later.
 */
/* @flow */

const deferred = []

 /**
  * Add a function to be executed, once the module finishes loading
  * @param {function} func
  */
export function defer(func :Function) :void {
	deferred.push(func)
}

 /**
  * Execute all stored deferred functions
  */
export function execDeferred() {
	while (deferred.length > 0) {
		deferred.shift()()
	}
}
