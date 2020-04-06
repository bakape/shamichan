use crate::util;
use protocol::debug_log;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use std::rc::Rc;
use yew::{
	agent::{Agent, AgentLink, Context, HandlerId},
	callback::Callback,
	Bridge, Bridged, Component, ComponentLink,
};

// Partial yew::Agent implementer wrapped by SubManager
pub trait PartialAgent: Default {
	// Data to be stored and delivered to subscribers
	type Data: Serialize + for<'de> Deserialize<'de> + Default + Clone;

	// Input message type
	type Input: Serialize + for<'de> Deserialize<'de>;

	// Update message type
	type Message;

	// Extra linking logic, like bridges
	#[allow(unused_variables)]
	fn init(
		&mut self,
		link: &AgentLink<SubManager<Self>>,
		data: &mut Self::Data,
	) -> util::Result {
		Ok(())
	}

	// Handle update message and return, if data was mutated
	#[allow(unused_variables)]
	fn update(
		&mut self,
		msg: Self::Message,
		link: &AgentLink<SubManager<Self>>,
		data: &mut Self::Data,
	) -> bool {
		false
	}

	// Handle input message and return, if data was mutated
	#[allow(unused_variables)]
	fn handle_input(
		&mut self,
		req: Self::Input,
		link: &AgentLink<SubManager<Self>>,
		data: &mut Self::Data,
	) -> bool {
		false
	}
}

// Message containing new value from SubManager
#[derive(Clone, Serialize, Deserialize)]
pub enum Message<T> {
	Initial(Rc<T>),
	Update { old: Rc<T>, new: Rc<T> },
}

// Extracts only the new value from the message
impl<T> Into<Rc<T>> for Message<T> {
	fn into(self) -> Rc<T> {
		match self {
			Message::Initial(v) => v,
			Message::Update { new, .. } => new,
		}
	}
}

// Wrapper type for implementing subscription-based data update propagation
pub struct SubManager<PA>
where
	PA: PartialAgent + 'static,
{
	link: AgentLink<Self>,
	clients: HashSet<HandlerId>,
	old_data: Rc<PA::Data>,
	new_data: PA::Data,
	inner: PA,
}

impl<PA> SubManager<PA>
where
	PA: PartialAgent + 'static,
{
	// Apply change to data and send change notification to all subscribers
	fn apply_change(&mut self) {
		debug_log!("sending changes");

		let msg = Message::Update {
			old: {
				let mut mv = self.new_data.clone().into();
				std::mem::swap(&mut mv, &mut self.old_data);
				mv
			},
			new: self.new_data.clone().into(),
		};
		for id in self.clients.iter() {
			self.link.respond(*id, msg.clone());
		}
	}
}

impl<PA> Agent for SubManager<PA>
where
	PA: PartialAgent + 'static,
{
	type Reach = Context;
	type Message = PA::Message;
	type Input = PA::Input;
	type Output = Message<PA::Data>;

	fn create(link: AgentLink<Self>) -> Self {
		let mut data = PA::Data::default();
		let mut inner = PA::default();
		util::log_error_res(inner.init(&link, &mut data));
		Self {
			link,
			inner,
			old_data: data.clone().into(),
			new_data: data,
			clients: HashSet::new(),
		}
	}

	fn update(&mut self, msg: Self::Message) {
		if self.inner.update(msg, &self.link, &mut self.new_data) {
			self.apply_change();
		}
	}

	fn handle_input(&mut self, req: Self::Input, _: HandlerId) {
		if self.inner.handle_input(req, &self.link, &mut self.new_data) {
			self.apply_change();
		}
	}

	fn connected(&mut self, id: HandlerId) {
		self.clients.insert(id);
		self.link
			.respond(id, Message::Initial(self.old_data.clone()));
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.clients.remove(&id);
	}
}

// Abstraction over AgentLink and ComponentLink
pub trait Link {
	type Output;

	fn make_callback<P, F>(&self, conv: F) -> Callback<Message<P::Data>>
	where
		P: PartialAgent,
		F: Fn(Message<P::Data>) -> Self::Output + 'static;
}

impl<A: Agent> Link for AgentLink<A> {
	type Output = A::Message;

	fn make_callback<PA, F>(&self, conv: F) -> Callback<Message<PA::Data>>
	where
		PA: PartialAgent,
		F: Fn(Message<PA::Data>) -> Self::Output + 'static,
	{
		debug_log!("creating agent callback");
		self.callback(conv)
	}
}

impl<C: Component> Link for ComponentLink<C> {
	type Output = C::Message;

	fn make_callback<PA, F>(&self, conv: F) -> Callback<Message<PA::Data>>
	where
		PA: PartialAgent,
		F: Fn(Message<PA::Data>) -> Self::Output + 'static,
	{
		debug_log!("creating component callback");
		self.callback(conv)
	}
}

// Helper for subscribing to and storing value provided by SubManager
pub struct Subscription<PA>
where
	PA: PartialAgent + 'static,
{
	// Maintains link
	#[allow(unused)]
	bridge: Box<dyn Bridge<SubManager<PA>>>,

	val: Rc<PA::Data>,
}

impl<PA> Subscription<PA>
where
	PA: PartialAgent + 'static,
{
	// Bridge SubManager with Component or Agent
	pub fn bridge<L, F>(link: &L, conv: F) -> Self
	where
		L: Link,
		F: Fn(Message<PA::Data>) -> L::Output + 'static,
	{
		Self {
			bridge: SubManager::bridge(link.make_callback::<PA, F>(conv)),
			val: PA::Data::default().into(),
		}
	}

	// Set a new value by consuming a message
	pub fn set(&mut self, msg: impl Into<Rc<PA::Data>>) {
		self.val = msg.into();
	}
}

impl<PA> std::ops::Deref for Subscription<PA>
where
	PA: PartialAgent + 'static,
{
	type Target = PA::Data;

	fn deref(&self) -> &Self::Target {
		self.val.deref()
	}
}

// Trait for cleaner subscription to a singleton value
pub trait Subscribe {
	type PA: PartialAgent;

	fn subscribe<L, F>(link: &L, conv: F) -> Subscription<Self::PA>
	where
		L: Link,
		F: Fn(Message<<Self::PA as PartialAgent>::Data>) -> L::Output + 'static,
	{
		debug_log!("subscribed");
		Subscription::<Self::PA>::bridge(link, conv)
	}
}
