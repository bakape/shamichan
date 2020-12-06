mod index;
mod thread;

pub use index::IndexFeed;
pub use thread::*;

use crate::{
	registry::{FeedClientSnapshot, Registry},
	util::Pulse,
};
use actix::prelude::*;
use common::payloads;
use std::{collections::HashSet, marker::PhantomData, time::Duration};

// TODO: use old Feed architecture from v7, except with a global feed that also
// receives all messages concerning the OP and last 5 posts
// TODO: keep track of clients that need init on the Feed itself
// TODO: merge FeedCommon and Feed as the global feed Actor will be very
// different from thread feed actors
// TODO: separate mutable and immutable pages on Feed. Store immutable pages in
// immutable memory mapped files.
// TODO: keep feed address on the client itself

/// Common functionality for thread and thread index feeds.
/// Must be embedded inside them.
#[derive(Debug)]
pub struct FeedCommon<F>
where
	F: Feed,
	F::Context: AsyncContext<F>,
{
	/// Pending processing of the current feed state
	pending_pulse: Option<SpawnHandle>,

	/// Link to the global registry
	registry: Addr<Registry>,

	/// Clients subscribed to the feed
	clients: FeedClientSnapshot,

	/// Clients that need init messages sent
	need_init: HashSet<u64>,

	/// To bind this to a concrete Feed implementor
	pd: PhantomData<F>,
}

impl<F> FeedCommon<F>
where
	F: Feed,
	F::Context: AsyncContext<F>,
{
	fn new(registry: Addr<Registry>) -> Self {
		Self {
			registry,
			clients: Default::default(),
			pending_pulse: Default::default(),
			need_init: Default::default(),
			pd: Default::default(),
		}
	}

	/// Schedule processing of the buffered state in 100ms, if not yet scheduled
	fn schedule_pulse(&mut self, ctx: &mut <F as Actor>::Context) {
		if self.pending_pulse.is_none() {
			self.pending_pulse =
				ctx.notify_later(Pulse, Duration::from_millis(100)).into();
		}
	}
}

/// Implements common feed functionality for types that embed FeedCommon
pub trait Feed: Actor<Context = Context<Self>> + Handler<Pulse> {
	/// Return a reference to the contained FeedCommon
	fn get_feed_common(&mut self) -> &mut FeedCommon<Self>;

	/// Process any buffered changes
	fn process_pulse(&mut self);
}

/// Implement common feed functionality for implementors of feed
///
/// Needed, because you can't simply implement a trait for all implementations
/// of a trait in rust. Specialization when?
#[macro_export]
macro_rules! impl_feed_commons {
	($dst:ident) => {
		impl actix::Actor for $dst {
			type Context = actix::Context<Self>;
		}

		impl actix::Handler<$crate::util::Pulse> for $dst {
			type Result = ();

			fn handle(
				&mut self,
				_: $crate::util::Pulse,
				_: &mut Self::Context,
			) -> Self::Result {
				self.get_feed_common().pending_pulse = None;
				self.process_pulse();
			}
		}

		impl actix::Handler<$crate::util::WakeUp> for $dst {
			type Result = ();

			fn handle(
				&mut self,
				_: $crate::util::WakeUp,
				ctx: &mut Self::Context,
			) -> Self::Result {
				self.get_feed_common().schedule_pulse(ctx);
			}
		}
	};
}

/// Either a thread or thread index feed
#[derive(Debug)]
pub enum AnyFeed {
	Index(Addr<IndexFeed>),
	Thread(Addr<ThreadFeed>),
}

impl AnyFeed {
	/// Notify a feed it has some external state changes to check.
	pub fn wake_up(&self) {
		match self {
			AnyFeed::Index(f) => f.do_send(crate::util::WakeUp),
			AnyFeed::Thread(f) => f.do_send(crate::util::WakeUp),
		};
	}
}

/// Insert a new post into a feed and return the matched feed
#[derive(Message, Debug)]
#[rtype(result = "()")]
pub struct InsertPost {
	pub id: u64,
	pub thread: u64,
	pub page: u32,
	pub opts: payloads::ReplyCreationOpts,
}

/// Insert a new thread into the thread index
#[derive(Message, Debug)]
#[rtype(result = "()")]
pub struct InsertThread {
	pub id: u64,
	pub subject: String,
	pub tags: Vec<String>,
	pub opts: payloads::PostCreationOpts,
}
