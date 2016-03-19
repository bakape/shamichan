import {SetMap} from './util'

type ActHandler = (arg?: any) => void
type ActMap = SetMap<ActHandler>

// Finite State Machine
export default class FSM<S, E> {
	private stateHandlers: ActMap = new SetMap<ActHandler>()
	private transitions: {[transition: string]: S} = {}
	private transitionHandlers: ActMap = new SetMap<ActHandler>()
	state: S

	// Create a new finite state machine with the supplied start state
	constructor(start: S) {
		this.state = start
	}

	// Assign a handler to be execute on arrival to a new state
	on(state: S, handler: ActHandler) {
		this.stateHandlers.add(state as any, handler)
	}

	// Specify state transition and an optional handler to execute on it.
	// Any of starts[] + event -> result
	act(starts: S[], event: E, result: S, handler?: ActHandler) {
		for (let start of starts) {
			const trans = this.transitionString(start, event)
			this.transitions[trans] = result
			if (handler) {
				this.transitionHandlers.add(trans, handler)
			}
		}
	}

	// Generate a transition string representation
	private transitionString(start: S, event: E): string {
		return `${start}+${event}`
	}

	// Feed an event to the FSM
	feed(event: E, arg?: any) {
		const trans = this.transitionString(this.state, event)
		this.transitionHandlers.forEach(trans, fn => fn(arg))
		const result = this.transitions[trans]
		this.stateHandlers.forEach(result as any, fn => fn(arg))
		this.state = result
	}

	// Returns a function that executes FSM.prototype.feed with the suplied
	// argument
	feeder(event: E): (arg?: any) => void {
		return arg => this.feed(event, arg)
	}
}
