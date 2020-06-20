// Utilities for yew Agent implementors

use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use yew::agent::{Agent, AgentLink, HandlerId};

// Implement connected() and disconnected() for yew::Agent implementors
// embedding SingleSubscription
#[macro_export]
macro_rules! agent_single_sub {
	($output:ty) => {
		type Output = $output;

		fn connected(&mut self, id: HandlerId) {
			self.sub.add_subscriber(id);
		}

		fn disconnected(&mut self, id: HandlerId) {
			self.sub.remove_subscriber(id);
		}
	};
}

// Manges single value subscriptions
pub struct SingleSubscription<A, V>
where
	A: Agent<Output = V>,
	V: Serialize + for<'de> Deserialize<'de> + Eq + Clone,
{
	pub link: AgentLink<A>,

	// Current value
	value: V,

	// Active subscribers to connection state change
	subscribers: HashSet<HandlerId>,
}

impl<A, V> SingleSubscription<A, V>
where
	A: Agent<Output = V>,
	V: Serialize + for<'de> Deserialize<'de> + Eq + Clone,
{
	pub fn new(link: AgentLink<A>, value: V) -> Self {
		Self {
			link,
			value,
			subscribers: Default::default(),
		}
	}

	// Set new value and send it to all subscribers
	pub fn set_value(&mut self, new: V) {
		if self.value != new {
			self.value = new;
			for id in self.subscribers.iter() {
				self.send_current(*id);
			}
		}
	}

	// Add subscriber and send it the current value
	pub fn add_subscriber(&mut self, id: HandlerId) {
		self.subscribers.insert(id);
		self.send_current(id);
	}

	pub fn remove_subscriber(&mut self, id: HandlerId) {
		self.subscribers.remove(&id);
	}

	// Send current value to subscriber
	fn send_current(&self, subscriber: HandlerId) {
		self.link.respond(subscriber, self.value.clone())
	}

	// Returns currently stored value
	pub fn get_value(&self) -> &V {
		&self.value
	}
}
