mod threads;

use super::FetchFeedData;
use crate::{
	client::{Client, SendMessage, SendMessageBatch},
	message::Message,
	message::Message as Msg,
	mt_context::{AsyncHandler, MTContext},
	registry::Registry,
	util::{self, DynResult, Pulse, WakeUp},
};
use actix::prelude::*;
use async_trait::async_trait;
use common::{
	payloads::{post_body::Node, Post, ThreadWithPosts},
	Encoder, MessageType,
};
use serde::Serialize;
use std::{collections::HashMap, sync::Arc};
use threads::Threads;

/// Change to be applied to thread data
#[derive(Debug)]
pub enum Change {
	InsertPost(Post),
	SetBody {
		id: u64,
		body: Arc<Node>,
		close_post: bool,
	},
}

/// Set of buffered changes for a particular thread
#[derive(Debug)]
pub struct ChangeSet {
	/// Source feed of the change
	pub source_feed: u64,

	/// Concatenated messages to be sent to clients
	pub message: Message,

	/// Unencoded contents of the messages
	pub changes: Vec<Change>,
}

/// Feed for the thread index
#[derive(Debug)]
pub struct IndexFeed {
	/// Pending processing of buffered changes
	pending_pulse: bool,

	/// Last snapshot of Clients subscribed to the feed
	clients: super::Clients,

	/// Link to the global registry
	registry: Addr<Registry>,

	/// Thread data with cached init messages
	threads: Threads,

	/// Pending changes
	changes: Vec<ChangeSet>,

	/// Global message encoder
	enc: Option<Encoder>,

	/// Fetches deferred to next pulse
	deferred_fetches: Vec<Addr<Client>>,
}

impl actix::Actor for IndexFeed {
	type Context = MTContext<Self>;
}

#[async_trait]
impl AsyncHandler<Pulse> for IndexFeed {
	type Error = util::Err;

	async fn handle(
		&mut self,
		_: Pulse,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.pending_pulse = false;
		self.clients = self
			.registry
			.send(crate::registry::SnapshotClients(0))
			.await?;

		// Send any deferred fetches before anything else to maintain chronology
		if !self.deferred_fetches.is_empty() {
			let msg = SendMessage(self.threads.get_message()?);
			for c in self.deferred_fetches.drain(0..) {
				c.do_send(msg.clone());
			}
		}

		let mut batch = vec![];
		if let Some(enc) = self.enc.take() {
			if !self.clients.is_empty() {
				batch.push(Msg::new(enc.finish()?));
			}
		}

		for cs in std::mem::take(&mut self.changes) {
			let t = match self.threads.get_mut(&cs.source_feed) {
				Some(t) => t,
				None => {
					// Handle feed messages arriving before a thread is inserted
					// by delaying it to the next pulse, that we schedule
					// immediately.
					//
					// This should almost never happen due to latency
					// differential, but still can.
					self.changes.push(cs);
					continue;
				}
			};
			if !self.clients.is_empty() {
				batch.push(cs.message);
			}
			for c in cs.changes {
				use Change::*;

				match c {
					InsertPost(p) => {
						t.thread.post_count += 1;
						t.posts.insert(p.id, p);
					}
					SetBody {
						id,
						body,
						close_post,
					} => {
						if let Some(p) = t.posts.get_mut(&id) {
							p.body = body;
							if close_post {
								p.open = false;
							}
						}
					}
				};
			}
		}

		if !batch.is_empty() {
			let batch = SendMessageBatch::new(batch);
			for c in self.clients.values() {
				c.do_send(batch.clone());
			}
		}

		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<WakeUp> for IndexFeed {
	type Error = ();

	async fn handle(
		&mut self,
		_: WakeUp,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.schedule_pulse(ctx);
		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<super::InsertThread> for IndexFeed {
	type Error = util::Err;

	/// This method is called for every message received by this actor.
	async fn handle(
		&mut self,
		msg: super::InsertThread,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		use common::payloads::{Post, Thread};

		self.schedule_pulse(ctx);

		let now = crate::util::now();
		let thread = ThreadWithPosts {
			thread: Thread::new(msg.id, now, msg.subject, msg.tags),
			posts: {
				let mut h = HashMap::new();
				h.insert(msg.id, Post::new_op(msg.id, now, msg.opts));
				h
			},
		};
		self.write_message(MessageType::InsertThread, &thread)?;
		self.threads.insert(msg.id, thread.into());

		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<ChangeSet> for IndexFeed {
	type Error = ();

	async fn handle(
		&mut self,
		changes: ChangeSet,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		self.schedule_pulse(ctx);
		self.changes.push(changes);
		Ok(())
	}
}

#[async_trait]
impl AsyncHandler<FetchFeedData> for IndexFeed {
	type Error = ();

	async fn handle(
		&mut self,
		FetchFeedData(client): FetchFeedData,
		ctx: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		match self.threads.get_cached_message() {
			Some(msg) => {
				client.do_send(SendMessage(msg));
			}
			None => {
				self.schedule_pulse(ctx);
				self.deferred_fetches.push(client);
			}
		};
		Ok(())
	}
}

/// Send set of used tags across all threads to client
pub struct UsedTags(pub Addr<Client>);

#[async_trait]
impl AsyncHandler<UsedTags> for IndexFeed {
	type Error = std::io::Error;

	async fn handle(
		&mut self,
		UsedTags(client): UsedTags,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		client.do_send(SendMessage(self.threads.used_tags()?));
		Ok(())
	}
}

impl IndexFeed {
	pub fn new(
		threads: Vec<ThreadWithPosts>,
		registry: Addr<Registry>,
	) -> Self {
		Self {
			registry,
			clients: Default::default(),
			pending_pulse: false,
			threads: Threads::new(
				threads
					.into_iter()
					.map(|t| (t.thread.id, t.into()))
					.collect(),
			),
			enc: Default::default(),
			changes: Default::default(),
			deferred_fetches: Default::default(),
		}
	}

	/// Schedule processing of buffered changes
	fn schedule_pulse(&mut self, ctx: &mut <Self as Actor>::Context) {
		if !self.pending_pulse {
			self.pending_pulse = true;
			ctx.notify_later(Pulse, super::PULSE_INTERVAL);
		}
	}

	/// Write message to pending message encoder
	pub fn write_message<T>(&mut self, t: MessageType, payload: &T) -> DynResult
	where
		T: Serialize + std::fmt::Debug,
	{
		common::log_msg_out!(t, payload);
		self.enc
			.get_or_insert_with(|| Default::default())
			.write_message(t, payload)?;
		Ok(())
	}
}
