type StateHandler = () => void

interface Stringable {
	toString(): string
}

// Finite State Machine
export default class FSM<S extends Stringable, E extends Stringable> {
	private stateHandlers: SetMap<StateHandler> = new SetMap<StateHandler>()
	private onceHandlers: SetMap<StateHandler> = new SetMap<StateHandler>()
	private changeHandlers: (() => void)[] = []
	private transitions: { [transition: string]: () => S } = {};
	private wilds: { [event: string]: () => S } = {};
	public state: S

	private feeding: boolean = false;
	private buffered: E[] = [];

	// Create a new finite state machine with the supplied start state
	constructor(start: S) {
		this.state = start
	}

	// Assign a handler to be execute on arrival to a new state
	public on(state: S, handler: StateHandler) {
		this.stateHandlers.add(state.toString(), handler)
	}

	// Like on, but handler is removed after execution
	public once(state: S, handler: StateHandler) {
		this.onceHandlers.add(state.toString(), handler)
	}

	// Add a handler for any state change
	public onChange(fn: () => void) {
		this.changeHandlers.push(fn);
	}

	// Specify state transition and a handler to execute on it. The handler must
	// return the next state of FSM.
	public act(start: S, event: E, handler: (arg?: any) => S) {
		this.transitions[this.transitionString(start, event)] = handler
	}

	// Specify an event and handler, that will execute, when this event is fired
	// on any state
	public wildAct(event: E, handler: (arg?: any) => S) {
		this.wilds[event.toString()] = handler
	}

	// Generate a transition string representation
	private transitionString(start: S, event: E): string {
		return `${start}+${event}`
	}

	// Feed an event to the FSM
	public feed(event: E) {
		// Ensure fed events are still sequential, even if feed() ends up
		// calling feed() down the call stack.
		if (this.feeding) {
			this.buffered.push(event);
			return;
		}
		this.feeding = true;

		let result: S
		const e = event.toString()
		if (e in this.wilds) {
			result = this.wilds[e]()
		} else {
			const transition = this.transitionString(this.state, event),
				handler = this.transitions[transition]
			if (!handler) { // Not registered. NOP
				return this.feedBuffered();
			}
			result = handler();
		}
		if (this.state === result) {
			return this.feedBuffered();
		}

		const r = result.toString();

		// These may depend on the previous state of the FSM
		this.onceHandlers.forEach(r, fn =>
			fn());
		this.onceHandlers.removeAll(r);

		// But stateHandlers must apply changes according to the new state
		this.state = result;
		this.stateHandlers.forEach(r, fn =>
			fn());
		for (let fn of this.changeHandlers) {
			fn();
		}

		this.feedBuffered();
	}

	// Feed any buffered events
	private feedBuffered() {
		this.feeding = false;
		if (this.buffered.length) {
			this.feed(this.buffered.shift());
		}
	}

	// Returns a function that executes FSM.prototype.feed with the passed
	// argument
	public feeder(event: E): StateHandler {
		return () =>
			this.feed(event);
	}
}

// Simple map of sets with automatic array creation and removal
class SetMap<V> {
	private map: { [key: string]: Set<V> } = {}

	// Add item to key
	public add(key: string, item: V) {
		if (!(key in this.map)) {
			this.map[key] = new Set()
		}
		this.map[key].add(item)
	}

	// Remove an item from a key
	public remove(key: string, item: V) {
		const set = this.map[key]
		if (!set) {
			return
		}
		set.delete(item)
		if (set.size === 0) {
			delete this.map[key]
		}
	}

	// Remove all items from a key
	public removeAll(key: string) {
		delete this.map[key]
	}

	// Execute a function for each item under a key
	public forEach(key: string, fn: (item: V) => void) {
		const set = this.map[key]
		if (!set) {
			return
		}
		set.forEach(fn)
	}
}
