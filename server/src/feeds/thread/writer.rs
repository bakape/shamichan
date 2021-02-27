use super::{
	super::index::{Change, ChangeSet, IndexFeed},
	last_5::Last5,
};
use crate::{
	client::Client, message::Message, mt_context::MTAddr, util::DynResult,
};
use actix::Addr;
use common::{Encoder, MessageType};
use serde::Serialize;

/// Pending messages and changes to be sent to the global thread index feed
#[derive(Default, Debug)]
struct Global {
	/// Messages to be encoded and sent
	enc: Encoder,

	/// Changes reflected by the messages
	changes: Vec<Change>,
}

/// Buffering writer of messages and feed changes.
///
/// Separation enables to limit mutable borrow scope.
#[derive(Debug)]
pub struct Writer {
	/// Feed ID
	feed: u64,

	/// Link to the thread index ThreadFeed
	index_feed: MTAddr<IndexFeed>,

	/// Last 5 post IDs in the thread
	last_5_posts: Last5,

	/// Pending message streaming encoder
	enc: Option<Encoder>,

	/// Pending messages and changes to be sent to the global thread index
	/// feed
	global: Option<Global>,
}

impl Writer {
	/// Construct new Writer for feed
	///
	/// `last_5_posts` can contain more than just the IDs of the last 5 posts.
	pub fn new(
		feed: u64,
		index_feed: MTAddr<IndexFeed>,
		last_5_posts: impl IntoIterator<Item = u64>,
	) -> Self {
		let mut s = Self {
			feed,
			index_feed,
			last_5_posts: Last5::new(feed),
			enc: Default::default(),
			global: Default::default(),
		};

		for id in last_5_posts {
			s.last_5_posts.push(id);
		}

		s
	}

	/// Register a new post ID in the thread
	#[inline]
	pub fn register_post_id(&mut self, id: u64) {
		self.last_5_posts.push(id);
	}

	/// Write post-related message to pending message encoder and propagate it
	/// to the global feed together with `change`, if needed.
	pub fn write_post_message<T>(
		&mut self,
		post_id: u64,
		t: MessageType,
		payload: &T,
		change: Change,
	) -> DynResult
	where
		T: Serialize + std::fmt::Debug,
	{
		self.write_message(t, payload)?;
		if post_id == self.feed || post_id >= self.last_5_posts.min() {
			self.write_global_change(t, payload, change)?;
		}
		Ok(())
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

	/// Write message and change to the pending global feed changeset
	pub fn write_global_change(
		&mut self,
		t: MessageType,
		payload: &impl Serialize,
		change: Change,
	) -> DynResult {
		let set = match &mut self.global {
			Some(s) => s,
			None => {
				self.global = Some(Default::default());
				self.global.as_mut().unwrap()
			}
		};
		set.enc.write_message(t, payload)?;
		set.changes.push(change);
		Ok(())
	}

	/// Flush changes and send them to all clients and the global feed
	pub fn flush<'a>(
		&mut self,
		clients: impl ExactSizeIterator<Item = &'a Addr<Client>>,
	) -> DynResult {
		if let Some(enc) = self.enc.take() {
			if clients.len() != 0 {
				let msg = Message::new(enc.finish()?);
				for c in clients {
					c.do_send(crate::client::SendMessage(msg.clone()));
				}
			}
		}
		if let Some(set) = self.global.take() {
			self.index_feed.do_send(ChangeSet {
				source_feed: self.feed,
				message: Message::new(set.enc.finish()?),
				changes: set.changes,
			})
		}
		Ok(())
	}
}
