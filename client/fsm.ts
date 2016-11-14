import {SetMap} from './util'

type StateHandler = (arg?: any) => void

// Finite State Machine
export default class FSM<S, E> {
	private stateHandlers: SetMap<StateHandler> = new SetMap<StateHandler>()
	private transitions: {[transition: string]: (arg?: any) => S} = {}
	private wilds: {[event: string]: (arg?: any) => S} = {}
	public state: S

	// Create a new finite state machine with the supplied start state
	constructor(start: S) {
		this.state = start
	}

	// Assign a handler to be execute on arrival to a new state
	public on(state: S, handler: StateHandler) {
		this.stateHandlers.add(state as any, handler)
	}

	// Specify state transition and a handler to execute on it. The handler must
	// return the next state of FSM.
	public act(start: S, event: E, handler: (arg?: any) => S) {
		this.transitions[this.transitionString(start, event)] = handler
	}

	// Specify an event and handler, that will execute, when this event is fired
	// on any state.
	public wildAct(event: E, handler: (arg?: any) => S) {
		this.wilds[event as any] = handler
	}

	// Generate a transition string representation
	private transitionString(start: S, event: E): string {
		return `${start}+${event}`
	}

	// Feed an event to the FSM
	public feed(event: E, arg?: any) {
		let result: S
		if (event as any in this.wilds) {
			result = this.wilds[event as any](arg)
		} else {
			const transition = this.transitionString(this.state, event),
				handler = this.transitions[transition]
			if (!handler) { // Not registered. NOOP
				return
			}
			result = handler(arg)
		}
		this.state = result
		this.stateHandlers.forEach(result as any, fn =>
			fn(arg))
	}

	// Returns a function that executes FSM.prototype.feed with the passed
	// argument
	public feeder(event: E): StateHandler {
		return arg =>
			this.feed(event, arg)
	}
}
