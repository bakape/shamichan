import {SetMap} from './common'

type ActHandler = (arg?: any) => void
type ActMap = SetMap<string, ActHandler>

// Finite State Machine
export default class FSM {
	private stateHandlers: ActMap
	private transitions: {[transition: string]: string}
	private transitionHandlers: ActMap
	state: string

	// Create a new finite state machine with the supplied start state
	constructor(start: string) {
		this.state = start
		this.stateHandlers = new SetMap<string, ActHandler>()
		this.transitionHandlers = new SetMap<string, ActHandler>()
	}

	// Assign a handler to be execute on arrival to a new state
	on(state: string, handler: ActHandler) {
		this.stateHandlers.add(state, handler)
	}

	// Specify state transition and an optional handler to execute on it.
	// Any of starts[] + added -> result
	act(starts: string[], added: string, result: string, handler?: ActHandler) {
		for (let start of starts) {
			const trans = this.transitionString(start, added)
			this.transitions[trans] = result
			if (handler) {
				this.transitionHandlers.add(trans, handler)
			}
		}
	}

	// Generate a transition string representation
	private transitionString(start: string, added: string): string {
		return `${start}+${added}`
	}

	// Transition the FSM to a new state
	feed(state: string, arg?: any) {
		this.stateHandlers.forEach(state, fn => fn(arg))
		const trans = this.transitionString(this.state, state)
		this.transitionHandlers.forEach(trans, fn => fn(arg))
		this.state = this.transitions[trans]
	}

	// Returns a function that executes FSM.prototype.feed with the suplied
	// argument
	feeder(state: string): (arg?: any) => void {
		return arg => this.feed(state, arg)
	}
}
