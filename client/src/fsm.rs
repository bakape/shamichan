use super::state::State;
use std::collections::{HashMap, VecDeque};
use std::hash::Hash;

type VecMap<K, V> = HashMap<K, Vec<V>>;

// Configurable Finite State Machine.
// Enables defining transitions from a set of states and various handlers for
// those.
//
// It is recommended for both S and E to be enums represented by the smallest
// integer capable of storing all of its variants.
//
// S: State type
// E: Event type
#[derive(Default)]
pub struct FSM<S: Eq + Hash + Default + Copy, E: Eq + Hash + Default + Copy> {
	// Run, when a state is reached
	on_state_handlers: VecMap<S, fn(&mut State)>,

	// Run, when a state is reached and then removed
	on_state_once_handlers: VecMap<S, fn(&mut State)>,

	// Run on any state change
	any_change_handlers: Vec<fn(&mut State)>,

	// Run, when an event fires on any state
	event_handlers: VecMap<E, fn(&mut State)>,

	// Run, when an event fires on a particular state.
	// Returns the next state of the FSM.
	transition_handlers: HashMap<(S, E), fn(&mut State, S, E) -> S>,

	// Run, when an event fires on any state.
	// Returns the next state of the FSM.
	any_state_transition_handlers: HashMap<E, fn(&mut State, S, E) -> S>,

	// Current state of the FSM
	state: S,

	// Event is currently being fed into the FSM
	is_feeding: bool,

	// Buffered events to be fired
	buffered: VecDeque<E>,
}

impl<S: Eq + Hash + Default + Copy, E: Eq + Hash + Default + Copy> FSM<S, E> {
	// Create new FSM with a given start state
	pub fn new(state: S) -> FSM<S, E> {
		Self {
			state: state,
			..Default::default()
		}
	}

	// Assign a handler to be execute on arrival to a new state
	pub fn on(&mut self, state: S, handler: fn(&mut State)) {
		self.on_state_handlers
			.entry(state)
			.or_default()
			.push(handler)
	}

	// Like on, but handler is removed after execution
	pub fn once(&mut self, state: S, handler: fn(&mut State)) {
		self.on_state_once_handlers
			.entry(state)
			.or_default()
			.push(handler)
	}

	// Specify source state and event sets that transition the FSM into another
	// state by calling handler.
	//
	// The handler receives the current FSM state and fired event and must
	// return the next state of the FSM.
	pub fn set_transitions(
		&mut self,
		states: &[S],
		events: &[E],
		handler: fn(&mut State, S, E) -> S,
	) {
		for s in states {
			for e in events {
				self.transition_handlers.insert((*s, *e), handler);
			}
		}
	}

	// Specify event set that transitions the FSM into another state by calling
	// handler.
	//
	// The handler receives the current FSM state and fired event and must
	// return the next state of the FSM.
	pub fn set_any_state_transitions(
		&mut self,
		events: &[E],
		handler: fn(&mut State, S, E) -> S,
	) {
		for e in events {
			self.any_state_transition_handlers.insert(*e, handler);
		}
	}

	// Feed an event into the FSM
	pub fn feed(&mut self, app_state: &mut State, event: E) {
		// Ensure fed events are still sequential, even if feed() ends up
		// calling feed() down the call stack.
		if self.is_feeding {
			self.buffered.push_back(event);
			return;
		}
		self.is_feeding = true;

		if let Some(handler) = self
			.any_state_transition_handlers
			.get(&event)
			.or_else(|| self.transition_handlers.get(&(self.state, event)))
		{
			self.state = handler(app_state, self.state, event);

			if let Some(handlers) =
				self.on_state_once_handlers.remove(&self.state)
			{
				for h in handlers {
					h(app_state);
				}
			}
			if let Some(handlers) = self.on_state_handlers.get(&self.state) {
				for h in handlers {
					h(app_state);
				}
			}
		}

		// Feed next buffered event, if any
		self.is_feeding = false;
		if let Some(event) = self.buffered.pop_front() {
			self.feed(app_state, event);
		}
	}

	// Return current state of FSM
	pub fn state(&self) -> S {
		self.state
	}
}

#[test]
fn basic_operation() {
	let mut fsm: FSM<u8, u8> = FSM::new(0);
	let mut app_state = State::default();

	fn handle(_app_state: &mut State, state: u8, event: u8) -> u8 {
		assert_eq!(state, 0);
		assert_eq!(event, 1);
		3
	}

	fsm.set_transitions(&[0], &[1], handle);
	fsm.feed(&mut app_state, 1);
	assert_eq!(fsm.state(), 3);
}
