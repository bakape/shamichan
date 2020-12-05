use crate::{
	body::persist_open::BodyFlusher,
	client::Client,
	feeds::{self, AnyFeed, IndexFeed, ThreadFeed},
	util,
};
use actix::prelude::*;
use common::{
	payloads::{Thread, ThreadWithPosts},
	util::SetMap,
};
use std::{collections::HashMap, sync::Arc};

// TODO: remove clients with disconnected addresses on send

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

/// Snapshot of all clients contained in a fedd
#[derive(Debug, Clone, Default)]
pub struct FeedClientSnapshot {
	/// Incremented on any modification to clients for cheap comparison
	pub update_counter: usize,

	/// The collection is wrapped in Arc to reduce snapshoting overhead as
	/// these are not likely to change as often as a feed demands a snapshot.
	pub clients: Arc<HashMap<u64, Addr<Client>>>,
}

impl FeedClientSnapshot {
	/// Set the contents of the snapshot
	fn set(&mut self, clients: HashMap<u64, Addr<Client>>) {
		self.update_counter += 1;
		self.clients = clients.into();
	}

	/// Modify the contents of the snapshot using f
	fn modify(&mut self, f: impl FnOnce(&mut HashMap<u64, Addr<Client>>)) {
		let mut new = (*self.clients).clone();
		f(&mut new);
		self.set(new);
	}
}

/// Keeps state and feed subscription of all clients
#[derive(Debug)]
pub struct Registry {
	/// All currently connected clients
	clients: HashMap<u64, ClientDescriptor>,

	/// Maps feed ID to clients that are synced to that feed
	feed_clients: HashMap<u64, FeedClientSnapshot>,

	/// Maps client public key ID to a set of clients using that ID
	by_pub_key: SetMap<u64, u64>,

	/// Thread index feed
	index_feed: Addr<IndexFeed>,

	/// Batching open post body flusher
	body_flusher: Addr<BodyFlusher>,

	/// All thread feeds in the system. One per existing thread.
	feeds: HashMap<u64, Addr<ThreadFeed>>,
}

impl Actor for Registry {
	type Context = actix::Context<Self>;
}

impl Registry {
	/// Initialize Registry instance by reading feed data from the database
	pub fn new(
		ctx: &mut Context<Self>,
		mut threads: Vec<ThreadWithPosts>,
	) -> Self {
		let feed_init: Vec<_> = threads
			.iter_mut()
			.map(|t| {
				(
					t.thread_data.clone(),
					t.posts.iter().map(|p| p.id).collect::<Vec<u64>>(),
				)
			})
			.collect();
		let index_feed = IndexFeed::new(threads, ctx.address()).start();
		let body_flusher = BodyFlusher::start_default();

		Self {
			clients: Default::default(),
			feed_clients: Default::default(),
			by_pub_key: Default::default(),
			index_feed: index_feed.clone(),
			body_flusher: body_flusher.clone(),
			feeds: feed_init
				.into_iter()
				.map(move |(t, last_5)| {
					(
						t.id,
						ThreadFeed::create(|tf_ctx| {
							ThreadFeed::new(
								tf_ctx,
								t,
								last_5,
								None,
								ctx.address(),
								index_feed.clone(),
								body_flusher.clone(),
							)
						}),
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
		self.clients
			.get_mut(id)
			.ok_or_else(|| format!("client not found: {}", id))
	}

	/// Get address of feed or return error
	fn get_feed(&self, id: &u64) -> Result<Addr<ThreadFeed>, String> {
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
				if let Some(snapshot) = self.feed_clients.get_mut(&feed) {
					snapshot.modify(|s| {
						s.remove(&client);
					});
				}
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
		let desc = self.get_client(&msg.client)?;

		#[rustfmt::skip]
		macro_rules! get_feed {
			($id:expr) => {
				if $id == &0 {
					AnyFeed::Index(self.index_feed.clone())
				} else {
					AnyFeed::Thread(self.get_feed($id)?)
				}
			};
		}

		if let Some(old_feed) = &desc.feed {
			if old_feed == &msg.feed {
				let old = *old_feed;
				return Ok(get_feed!(&old));
			}
			if let Some(feed) = desc.feed.take() {
				if let Some(snapshot) = self.feed_clients.get_mut(&feed) {
					snapshot.modify(|s| {
						s.remove(&msg.client);
					});
				}
			}
		}

		let new_feed = get_feed!(&msg.feed);
		new_feed.wake_up();
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
#[rtype(result = "Result<Addr<ThreadFeed>, String>")]
pub struct GetFeed(pub u64);

impl Handler<GetFeed> for Registry {
	type Result = Result<Addr<ThreadFeed>, String>;

	fn handle(
		&mut self,
		GetFeed(id): GetFeed,
		_: &mut Self::Context,
	) -> Self::Result {
		self.get_feed(&id)
	}
}

/// Create a Feed instance for a new thread
#[derive(Message)]
#[rtype(result = "Addr<ThreadFeed>")]
pub struct InsertThread(pub feeds::InsertThread);

impl Handler<InsertThread> for Registry {
	type Result = Addr<ThreadFeed>;

	fn handle(
		&mut self,
		InsertThread(req): InsertThread,
		ctx: &mut Self::Context,
	) -> Self::Result {
		use common::payloads;

		let now = util::now();
		let addr = ThreadFeed::create(|tf_ctx| {
			ThreadFeed::new(
				tf_ctx,
				Thread::new(req.id, now, req.subject.clone(), req.tags.clone()),
				vec![req.id],
				Some(payloads::Page {
					thread: req.id,
					page: 0,
					posts: {
						let mut m = HashMap::new();
						m.insert(
							req.id,
							payloads::Post::new_op(
								req.id,
								now,
								req.opts.clone(),
							),
						);
						m
					},
				}),
				ctx.address(),
				self.index_feed.clone(),
				self.body_flusher.clone(),
			)
		});
		self.feeds.insert(req.id, addr.clone());

		// don't block registry on index feed
		self.index_feed.do_send(req);

		addr
	}
}

impl Handler<feeds::InsertPost> for Registry {
	type Result = ();

	fn handle(
		&mut self,
		req: feeds::InsertPost,
		ctx: &mut Self::Context,
	) -> Self::Result {
		todo!()
	}
}
