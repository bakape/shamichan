use crate::{
	body::persist_open::BodyFlusher,
	client::Client,
	feeds::{self, AnyFeed, IndexFeed, ThreadFeed},
	mt_context::{run, MTAddr},
	util::{self, SnapshotSource, WakeUp},
};
use actix::dev::{MessageResponse, ResponseChannel};
use actix::prelude::*;
use common::{
	payloads::{Thread, ThreadWithPosts},
	util::SetMap,
};
use std::collections::HashMap;

/// Stores client state and address
#[derive(Debug)]
struct ClientDescriptor {
	/// Specific feed a client is synchronized to.
	/// Is unset (default) - before the first sync message is received.
	feed: Option<u64>,

	/// The internal public key ID the client is registered with
	pub_key: Option<u64>,

	/// Address for communication
	addr: Addr<Client>,
}

/// Keeps state and feed subscription of all clients
#[derive(Debug)]
pub struct Registry {
	/// All currently connected clients
	clients: HashMap<u64, ClientDescriptor>,

	/// Maps feed ID to clients that are synced to that feed
	feed_clients: HashMap<u64, SnapshotSource<HashMap<u64, Addr<Client>>>>,

	/// Maps client public key ID to a set of clients using that ID
	by_pub_key: SetMap<u64, u64>,

	/// Thread index feed
	index_feed: MTAddr<IndexFeed>,

	/// Batching open post body flusher
	body_flusher: MTAddr<BodyFlusher>,

	/// All thread feeds in the system. One per existing thread.
	feeds: HashMap<u64, MTAddr<ThreadFeed>>,
}

impl Actor for Registry {
	type Context = Context<Self>;

	fn started(&mut self, ctx: &mut Self::Context) {
		// This is a central synchronization point.
		// The default of 16 is not enough.
		ctx.set_mailbox_capacity(1 << 10);
	}
}

/// Get reference to client or return error.
///
/// Macro, because Rust does not have partial self borrows in methods but it
/// does in inline code.
#[rustfmt::skip]
macro_rules! get_client {
	($self:expr, $id:expr) => {{
		$self.clients
		.get_mut($id)
		.ok_or_else(|| format!("client not found: {}", $id))
	}};
}

/// Notify a feed it ahs updates and should check them
///
/// Macro, because Rust does not have partial self borrows in methods but it
/// does in inline code.
#[rustfmt::skip]
macro_rules! wake_up_feed {
	($self:expr, $id:expr) => {
		if $id == 0 {
			$self.index_feed.do_send(WakeUp);
		} else {
			if let Some(f) = $self.feeds.get(&$id) {
				f.do_send(WakeUp);
			}
		}
	};
}

impl Registry {
	/// Initialize Registry instance by reading feed data from the database
	pub fn new(
		ctx: &mut Context<Self>,
		mut threads: Vec<ThreadWithPosts>,
	) -> Self {
		let feed_init_data: Vec<_> = threads
			.iter_mut()
			.map(|t| {
				(
					t.thread_data.clone(),
					t.posts.keys().copied().collect::<Vec<u64>>(),
				)
			})
			.collect();
		let index_feed = run(IndexFeed::new(threads, ctx.address()));
		let body_flusher = run(BodyFlusher::default());

		Self {
			clients: Default::default(),
			feed_clients: Default::default(),
			by_pub_key: Default::default(),
			index_feed: index_feed.clone(),
			body_flusher: body_flusher.clone(),
			feeds: feed_init_data
				.into_iter()
				.map(move |(t, last_5)| {
					(
						t.id,
						run(ThreadFeed::new(
							t,
							last_5,
							None,
							ctx.address(),
							index_feed.clone(),
							body_flusher.clone(),
						)),
					)
				})
				.collect(),
		}
	}

	/// Get reference to client or return error
	fn get_client(
		&mut self,
		id: &u64,
	) -> Result<&mut ClientDescriptor, String> {
		get_client!(self, id)
	}

	/// Notify a feed it ahs updates and should check them
	fn wake_up_feed(&self, id: u64) {
		wake_up_feed!(self, id);
	}

	/// Get address of a thread feed or return error
	fn get_thread_feed_addr(
		&self,
		id: &u64,
	) -> Result<MTAddr<ThreadFeed>, String> {
		Ok(self
			.feeds
			.get(id)
			.ok_or(format!("feed not found: {}", id))?
			.clone())
	}
}

/// Request to register a client
#[derive(Message)]
#[rtype(result = "()")]
pub struct RegisterClient {
	pub id: u64,
	pub addr: Addr<Client>,
}

impl Handler<RegisterClient> for Registry {
	type Result = ();

	fn handle(
		&mut self,
		msg: RegisterClient,
		_: &mut Self::Context,
	) -> Self::Result {
		self.clients.insert(
			msg.id,
			ClientDescriptor {
				feed: None,
				pub_key: None,
				addr: msg.addr,
			},
		);
	}
}

/// Remove client from registry by ID
#[derive(Message)]
#[rtype(result = "()")]
pub struct UnregisterClient(pub u64);

impl Handler<UnregisterClient> for Registry {
	type Result = ();

	fn handle(
		&mut self,
		UnregisterClient(client): UnregisterClient,
		_: &mut Self::Context,
	) -> Self::Result {
		// NOP, if client already removed
		if let Some(mut desc) = self.clients.remove(&client) {
			if let Some(feed) = desc.feed.take() {
				if let Some(s) = self.feed_clients.get_mut(&feed) {
					s.remove(&client);
				}
				self.wake_up_feed(feed);
			}
			if let Some(pub_key) = desc.pub_key {
				self.by_pub_key.remove(&pub_key, &client);
			}
		}
	}
}

/// Set client feed
#[derive(Message)]
#[rtype(result = "Result<AnyFeed, String>")]
pub struct SetFeed {
	pub client: u64,
	pub feed: u64,
}

impl Handler<SetFeed> for Registry {
	type Result = Result<AnyFeed, String>;

	fn handle(&mut self, msg: SetFeed, _: &mut Self::Context) -> Self::Result {
		use std::collections::hash_map::Entry::*;

		let new_feed = if msg.feed == 0 {
			AnyFeed::Index(self.index_feed.clone())
		} else {
			AnyFeed::Thread(self.get_thread_feed_addr(&msg.feed)?)
		};
		let desc = get_client!(self, &msg.client)?;

		// Clean up client registration on the old feed
		if let Some(old_feed) = &desc.feed {
			if old_feed == &msg.feed {
				// Nothing changed
				return Ok(new_feed);
			}
			if let Some(s) = self.feed_clients.get_mut(old_feed) {
				s.remove(&msg.client);
			}
			wake_up_feed!(self, *old_feed);
		}

		// Record client registration on the new feed
		desc.feed = msg.feed.into();
		match self.feed_clients.entry(msg.feed) {
			Occupied(mut e) => {
				e.get_mut().insert(msg.client, desc.addr.clone());
			}
			Vacant(e) => {
				e.insert(SnapshotSource::new({
					let mut h = HashMap::new();
					h.insert(msg.client, desc.addr.clone());
					h
				}));
			}
		}

		new_feed.wake_up();
		new_feed.do_send(feeds::FetchFeedData(desc.addr.clone()));
		Ok(new_feed)
	}
}

/// Set client public key
#[derive(Message)]
#[rtype(result = "Result<(), String>")]
pub struct SetPublicKey {
	pub client: u64,
	pub pub_key: u64,
}

impl Handler<SetPublicKey> for Registry {
	type Result = Result<(), String>;

	fn handle(
		&mut self,
		SetPublicKey { client, pub_key }: SetPublicKey,
		_: &mut Self::Context,
	) -> Self::Result {
		let desc = self.get_client(&client)?;
		desc.pub_key = Some(pub_key);
		self.by_pub_key.insert(pub_key, client);
		Ok(())
	}
}

/// Retrieve a ThreadFeed address from the registry
#[derive(Message)]
#[rtype(result = "Result<MTAddr<ThreadFeed>, String>")]
pub struct GetFeed(pub u64);

impl Handler<GetFeed> for Registry {
	type Result = Result<MTAddr<ThreadFeed>, String>;

	fn handle(
		&mut self,
		GetFeed(id): GetFeed,
		_: &mut Self::Context,
	) -> Self::Result {
		self.get_thread_feed_addr(&id)
	}
}

/// Create a Feed instance for a new thread
pub struct InsertThread(pub feeds::InsertThread);

impl Message for InsertThread {
	type Result = MTAddr<ThreadFeed>;
}

// Implemented here because it's not derivable
impl MessageResponse<Registry, InsertThread> for MTAddr<ThreadFeed> {
	fn handle<R: ResponseChannel<InsertThread>>(
		self,
		_: &mut <Registry as Actor>::Context,
		tx: Option<R>,
	) {
		if let Some(tx) = tx {
			tx.send(self);
		}
	}
}

impl Handler<InsertThread> for Registry {
	type Result = MTAddr<ThreadFeed>;

	fn handle(
		&mut self,
		InsertThread(req): InsertThread,
		ctx: &mut Self::Context,
	) -> Self::Result {
		use common::payloads;

		let now = util::now();
		let addr = run(ThreadFeed::new(
			Thread::new(req.id, now, req.subject.clone(), req.tags.clone()),
			None,
			Some(vec![payloads::Post::new_op(req.id, now, req.opts.clone())]),
			ctx.address(),
			self.index_feed.clone(),
			self.body_flusher.clone(),
		));
		self.feeds.insert(req.id, addr.clone());

		self.index_feed.do_send(req);

		addr
	}
}

/// Request a snapshot of the current clients of a feed
pub struct SnapshotClients(pub u64);

impl Message for SnapshotClients {
	type Result = feeds::Clients;
}

// Implemented here because it's not derivable
impl MessageResponse<Registry, SnapshotClients> for feeds::Clients {
	fn handle<R: ResponseChannel<SnapshotClients>>(
		self,
		_: &mut <Registry as Actor>::Context,
		tx: Option<R>,
	) {
		if let Some(tx) = tx {
			tx.send(self);
		}
	}
}

impl Handler<SnapshotClients> for Registry {
	type Result = feeds::Clients;

	fn handle(
		&mut self,
		req: SnapshotClients,
		_: &mut Self::Context,
	) -> Self::Result {
		self.feed_clients
			.get_mut(&req.0)
			.map(|s| s.snapshot())
			.unwrap_or_default()
	}
}

/// Returns the address of the IndexFeed
pub struct GetIndexFeed;

impl Message for GetIndexFeed {
	type Result = MTAddr<IndexFeed>;
}

// Implemented here because it's not derivable
impl MessageResponse<Registry, GetIndexFeed> for MTAddr<IndexFeed> {
	fn handle<R: ResponseChannel<GetIndexFeed>>(
		self,
		_: &mut <Registry as Actor>::Context,
		tx: Option<R>,
	) {
		if let Some(tx) = tx {
			tx.send(self);
		}
	}
}

impl Handler<GetIndexFeed> for Registry {
	type Result = MTAddr<IndexFeed>;

	fn handle(
		&mut self,
		_: GetIndexFeed,
		_: &mut Self::Context,
	) -> Self::Result {
		self.index_feed.clone()
	}
}
