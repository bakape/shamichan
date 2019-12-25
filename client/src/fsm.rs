use super::state::State;
use super::util;
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
pub struct FSM<S: Eq + Hash + Copy, E: Eq + Hash + Copy> {
	// Run, when a state is reached
	on_state_handlers: Vec<Box<dyn Fn(&mut State, S) -> util::Result>>,

	// Run, when a state is reached and then removed
	on_state_once_handlers: VecMap<S, Box<dyn Fn(&mut State) -> util::Result>>,

	// Run on any state change
	any_change_handlers: Vec<Box<dyn Fn(&mut State) -> util::Result>>,

	// Run, when an event fires on any state
	event_handlers: VecMap<E, Box<dyn Fn(&mut State) -> util::Result>>,

	// Run, when an event fires on a particular state.
	// Returns the next state of the FSM.
	transition_handlers:
		HashMap<(S, E), Box<dyn Fn(&mut State, S, E) -> util::Result<S>>>,

	// Run, when an event fires on any state.
	// Returns the next state of the FSM.
	any_state_transition_handlers:
		HashMap<E, Box<dyn Fn(&mut State, S, E) -> util::Result<S>>>,

	// Current state of the FSM
	state: S,

	// Event is currently being fed into the FSM
	is_feeding: bool,

	// Buffered events to be fired
	buffered: VecDeque<E>,
}

impl<S: Eq + Hash + Copy, E: Eq + Hash + Copy> FSM<S, E> {
	// Create new FSM with a given start state
	pub fn new(state: S) -> FSM<S, E> {
		Self {
			on_state_handlers: Vec::new(),
			on_state_once_handlers: HashMap::new(),
			any_change_handlers: Vec::new(),
			event_handlers: HashMap::new(),
			transition_handlers: HashMap::new(),
			any_state_transition_handlers: HashMap::new(),
			state: state,
			is_feeding: false,
			buffered: VecDeque::new(),
		}
	}

	// Assign a handler to be execute on arrival to a new state
	pub fn on_change<F>(&mut self, handler: &'static F)
	where
		F: Fn(&mut State, S) -> util::Result,
	{
		self.on_state_handlers.push(Box::from(handler))
	}

	// Execute handler and remove it after reaching a particular state
	pub fn once<F>(&mut self, state: S, handler: &'static F)
	where
		F: Fn(&mut State) -> util::Result,
	{
		self.on_state_once_handlers
			.entry(state)
			.or_default()
			.push(Box::from(handler))
	}

	// Specify source state and event sets that transition the FSM into another
	// state by calling handler.
	//
	// The handler receives the current FSM state and fired event and must
	// return the next state of the FSM.
	pub fn set_transitions<F>(
		&mut self,
		states: &[S],
		events: &[E],
		handler: &'static F,
	) where
		F: Fn(&mut State, S, E) -> util::Result<S>,
	{
		for s in states {
			for e in events {
				self.transition_handlers
					.insert((*s, *e), Box::from(handler));
			}
		}
	}

	// Specify event set that transitions the FSM into another state by calling
	// handler.
	//
	// The handler receives the current FSM state and fired event and must
	// return the next state of the FSM.
	pub fn set_any_state_transitions<F>(
		&mut self,
		events: &[E],
		handler: &'static F,
	) where
		F: Fn(&mut State, S, E) -> util::Result<S>,
	{
		for e in events {
			self.any_state_transition_handlers
				.insert(*e, Box::from(handler));
		}
	}

	// Feed an event into the FSM
	pub fn feed(&mut self, app_state: &mut State, event: E) -> util::Result {
		// Ensure fed events are still sequential, even if feed() ends up
		// calling feed() down the call stack.
		if self.is_feeding {
			self.buffered.push_back(event);
			return Ok(());
		}
		self.is_feeding = true;

		if let Some(handler) = self
			.any_state_transition_handlers
			.get(&event)
			.or_else(|| self.transition_handlers.get(&(self.state, event)))
		{
			self.state = handler(app_state, self.state, event)?;

			if let Some(handlers) =
				self.on_state_once_handlers.remove(&self.state)
			{
				for h in handlers {
					h(app_state)?;
				}
			}
			for h in self.on_state_handlers.iter() {
				h(app_state, self.state)?;
			}
		}

		// Feed next buffered event, if any
		self.is_feeding = false;
		if let Some(event) = self.buffered.pop_front() {
			self.feed(app_state, event)?;
		}
		Ok(())
	}

	// Return current state of FSM
	pub fn state(&self) -> S {
		self.state
	}
}

#[test]
fn basic_operation() -> super::util::Result {
	let mut fsm: FSM<u8, u8> = FSM::new(0);
	let mut app_state = State::default();

	fsm.set_transitions(&[0], &[1], &|_app_state, state, event| {
		assert_eq!(state, 0);
		assert_eq!(event, 1);
		Ok(3)
	});
	fsm.feed(&mut app_state, 1)?;
	assert_eq!(fsm.state(), 3);

	Ok(())
}
