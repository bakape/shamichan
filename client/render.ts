// Utility functions for reducing layout thrashing, by batching DOM writes and
// reads. Basically a stripped down version of FastDOM.

type Operation = () => void

let readStack: Operation[] = [],
	writeStack: Operation[] = [],
	scheduled: boolean

// Schedule a DOM write operation
export function write(operation: Operation) {
	writeStack.push(operation)
	scheduleFlush()
}

// Schedule a DOM read operation
export function read(operation: Operation) {
	readStack.push(operation)
	scheduleFlush()
}

// Schedule a flush on the next animation frame, if not yet scheduled
function scheduleFlush() {
	if (!scheduled) {
		scheduled = true
		requestAnimationFrame(flush)
	}
}

// Perform all write tasks and then read tasks in the stack
function flush() {
	for (let i = 0; i < writeStack.length; i++) {
		writeStack[i]()
	}
	for (let i = 0; i < readStack.length; i++) {
		readStack[i]()
	}
	writeStack = []
	readStack = []
	scheduled = false
}
