import {SetMap} from './util'

type ActHandler = (arg?: any) => void
type ActMap = SetMap<ActHandler>

// Finite State Machine
export default class FSM<S, E> {
	stateHandlers: ActMap = new SetMap<ActHandler>()
	transitions: {[transition: string]: S} = {}
	transitionHandlers: ActMap = new SetMap<ActHandler>()
	wilds: {[event: string]: S} = {}
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

	// Specify an event and optional handler, that will cause any state to
	// transition to the target result state.
	wildAct(event: E, result: S, handler?: ActHandler) {
		this.wilds[event as any] = result
		if (handler) {
			this.on(result, handler)
		}
	}

	// Generate a transition string representation
	transitionString(start: S, event: E): string {
		return `${start}+${event}`
	}

	// Feed an event to the FSM
	feed(event: E, arg?: any) {
		let result: S
		if (event as any in this.wilds) {
			result = this.wilds[event as any]
		} else {
			const trans = this.transitionString(this.state, event)
			this.transitionHandlers.forEach(trans, fn => fn(arg))
			result = this.transitions[trans]
		}
		this.stateHandlers.forEach(result as any, fn => fn(arg))
		this.state = result
	}

	// Returns a function that executes FSM.prototype.feed with the suplied
	// argument
	feeder(event: E): (arg?: any) => void {
		return arg => this.feed(event, arg)
	}
}
