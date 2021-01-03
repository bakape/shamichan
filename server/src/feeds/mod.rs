mod index;
mod thread;

pub use index::IndexFeed;
pub use thread::*;

use crate::{
	client::Client,
	mt_context::{AsyncHandler, MTAddr},
	util::Snapshot,
};
use actix::prelude::*;
use common::payloads;
use std::collections::HashMap;

/// Shorthand for a snapshot of clients
pub type Clients = Snapshot<HashMap<u64, Addr<Client>>>;

/// Either a thread or thread index feed
#[derive(Debug)]
pub enum AnyFeed {
	Index(MTAddr<IndexFeed>),
	Thread(MTAddr<ThreadFeed>),
}

impl AnyFeed {
	/// Notify a feed it has some external state changes to check.
	pub fn wake_up(&self) {
		self.do_send(crate::util::WakeUp);
	}

	/// Send a request to a a feed, if both feed kinds support it
	pub fn do_send<R>(&self, req: R)
	where
		R: Send + 'static,
		IndexFeed: AsyncHandler<R>,
		ThreadFeed: AsyncHandler<R>,
	{
		match self {
			AnyFeed::Index(f) => f.do_send(req),
			AnyFeed::Thread(f) => f.do_send(req),
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

/// Send thread metainformation for thread feeds or thread catalog for index
/// feeds
pub struct FetchFeedData(pub Addr<Client>);
