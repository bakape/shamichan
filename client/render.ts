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
	const writes = writeStack,
		reads = readStack
	writeStack = []
	readStack = []
	for (let i = 0; i < writes.length; i++) {
		writes[i]()
	}
	for (let i = 0; i < reads.length; i++) {
		reads[i]()
	}
	scheduled = false
}
