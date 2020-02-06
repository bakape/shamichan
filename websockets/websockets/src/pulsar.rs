use super::common::DynResult;
use super::{bindings, registry};
use protocol::*;
use rayon::prelude::*;
use serde::Serialize;
use std::collections::{HashMap, HashSet};
use std::sync::{
	mpsc::{channel, SendError, Sender},
	Arc, Mutex,
};
use std::time::{Duration, Instant, SystemTime};

// TODO: Optimise global feed with partial init message caching
// TODO: Ensure client does not try to mutate feed data before it has been
// initialized Those NOP on the server.
// TODO: Add feed data on thread creation

// For sending requests to Pulsar. Clone to use.
static mut REQUEST: Option<Mutex<Sender<Request>>> = None;

// Init module state
pub fn init(feed_data: &[u8]) -> DynResult {
	let (sdr, recv) = channel();
	unsafe {
		REQUEST = Some(Mutex::new(sdr));
	}
	let mut p: Pulsar = Default::default();
	p.init(feed_data)?;
	std::thread::Builder::new()
		.name("pulsar".into())
		.spawn(move || {
			const SEND_INTERVAL: Duration = Duration::from_millis(100);
			const CLEANUP_INTERVAL: Duration = Duration::from_secs(10);

			let now = Instant::now();
			let mut last_send = now;
			let mut last_cleanup = now;

			loop {
				let started = Instant::now();

				// Process all pending requests
				for req in recv.try_iter() {
					match req {
						Request::CreateThread(data) => p.create_thread(data),
						Request::RemoveThread(id) => p.remove_thread(id),
						Request::InsertImage(req) => p.insert_image(req),
					}
				}

				if started - last_send > SEND_INTERVAL {
					last_send = now;

					// Block until messages are sent to the Go side to guarantee
					// sequentiality
					p.send_messages();
				}
				if started - last_cleanup > CLEANUP_INTERVAL {
					last_cleanup = now;
					p.clean_up();
				}

				// Sleep thread to save resources.
				// Compensate for a possibly long tick.
				let elapsed = Instant::now() - started;
				let mut dur = Duration::from_millis(10);
				// Duration can not be negative
				if elapsed < dur {
					dur -= elapsed;
				}
				if dur.as_millis() != 0 {
					std::thread::sleep(dur);
				}
			}
		})?;
	Ok(())
}

// Holds the IDs of up to the last 5 posts
type Last5Posts =
	heapless::BinaryHeap<u64, heapless::consts::U5, heapless::binary_heap::Min>;

// Common to both thread feeds and the global Feed
#[derive(Default, Debug)]
struct FeedCommon {
	// Thread ID or 0 for global feed
	id: u64,

	// Clients needing an init message sent
	need_init: HashSet<u64>,

	// Cached encoded initialization message buffer
	init_msg_cache: Option<Msg>,

	// Pending message streaming encoder
	pending: Option<Encoder>,
}

impl FeedCommon {
	// Clear all cached values
	fn clear_cache(&mut self) {
		self.init_msg_cache = None;
	}

	// This should never happen, but log it and halt execution, if it does.
	// Caller should abort execution.
	fn log_encode_error(&self, err: std::io::Error) {
		bindings::log_error(&format!(
			"could not encode feed data: feed_id={} err={:?}",
			self.id, err
		));
	}
}

// Update feed. Either a thread feed or the global thread index feed.
#[derive(Default, Debug)]
struct Feed {
	common: FeedCommon,

	global_init_msg_part: Option<Msg>,

	// Pending messages for global thread index feed
	pending_global: Option<Encoder>,

	// Last 5 post IDs in thread
	last_5_posts: Last5Posts,

	// Current active feed data.
	data: FeedData,
}

impl Feed {
	// Create new wrapped Feed initialized with data
	fn new(data: FeedData) -> Self {
		// Find last 5 posts added to thread
		let mut l5 = Last5Posts::default();
		for id in data.recent_posts.keys() {
			if match l5.peek() {
				Some(min) => {
					if min < id {
						l5.pop();
						true
					} else {
						false
					}
				}
				None => true,
			} {
				unsafe { l5.push_unchecked(*id) };
			}
		}

		Self {
			common: FeedCommon {
				id: data.thread,
				..Default::default()
			},
			last_5_posts: l5,
			data: data,
			..Default::default()
		}
	}

	// Clear all cached values
	fn clear_cache(&mut self) {
		self.global_init_msg_part = None;
		self.common.clear_cache()
	}

	// Encode and cache feed init message or return cached one.
	fn get_init_msg(&mut self) -> std::io::Result<Msg> {
		match &mut self.common.init_msg_cache {
			Some(msg) => Ok(msg.clone()),
			None => {
				let msg = Msg::new({
					let mut enc = Encoder::new(Vec::new());
					enc.write_message(MessageType::FeedInit, &self.data)?;
					enc.finish()?
				});
				self.common.init_msg_cache = Some(msg.clone());
				Ok(msg)
			}
		}
	}

	// Return, if post should be included in global thread index
	fn include_in_global(&self, id: u64) -> bool {
		id == self.data.thread || self.last_5_posts.iter().any(|x| id == *x)
	}

	// Encode and cache global feed init message part or return cached one.
	fn get_global_init_msg_part(&mut self) -> std::io::Result<Msg> {
		match &mut self.global_init_msg_part {
			Some(msg) => Ok(msg.clone()),
			None => {
				let msg = Msg::new({
					let mut enc = Encoder::new(Vec::new());

					macro_rules! filter_recent {
						($key:ident) => {
							self.data
								.$key
								.iter()
								.filter(|(id, _)| self.include_in_global(**id))
								.map(|(k, v)| (*k, v.clone()))
								.collect()
						};
					}
					let res = rayon::join(
						|| filter_recent!(recent_posts),
						|| filter_recent!(open_posts),
					);

					enc.write_message(
						MessageType::FeedInit,
						&FeedData {
							thread: self.common.id,
							recent_posts: res.0,
							open_posts: res.1,
						},
					)?;
					enc.finish()?
				});
				self.global_init_msg_part = Some(msg.clone());
				Ok(msg)
			}
		}
	}

	// Insert new blank open post
	fn insert_post(&mut self, id: u64) {
		let now = SystemTime::now()
			.duration_since(SystemTime::UNIX_EPOCH)
			.expect("negative Unix timestamp")
			.as_secs();

		self.data.recent_posts.insert(id, now);
		if self.last_5_posts.len() == 5 {
			unsafe { self.last_5_posts.pop_unchecked() };
		}
		unsafe { self.last_5_posts.push_unchecked(id) };

		self.data.open_posts.insert(
			id,
			OpenPost {
				thread: self.common.id,
				created_on: now,
				..Default::default()
			},
		);
	}

	// Get or init new Encoder and return it
	fn get_encoder(enc: &mut Option<Encoder>) -> &mut Encoder {
		match enc {
			Some(e) => e,
			None => {
				*enc = Some(Encoder::new(vec![]));
				enc.as_mut().unwrap()
			}
		}
	}

	// Write post-related message to thread and possibly global feed
	fn encode_post_message(
		&mut self,
		post: u64,
		typ: MessageType,
		payload: &impl Serialize,
	) {
		#[rustfmt::skip]
		macro_rules! encode {
			($dst:expr) => {
				if $dst.is_none() {
					$dst = Some(Encoder::new(Vec::new()));
				}
				if let Err(err) =
					Self::get_encoder(&mut $dst).write_message(typ, payload)
				{
					self.common.log_encode_error(err);
				}
			};
		}

		encode!(self.common.pending);
		if self.include_in_global(post) {
			encode!(self.pending_global);
		}
	}
}

// Reusable message buffer wrapper with AsRef[u8]
#[derive(Clone, Debug)]
struct Msg(Arc<Vec<u8>>);

impl Msg {
	fn new(buf: Vec<u8>) -> Self {
		Arc::new(buf).into()
	}
}

impl AsRef<[u8]> for Msg {
	fn as_ref(&self) -> &[u8] {
		self.0.as_slice().as_ref()
	}
}

impl From<Arc<Vec<u8>>> for Msg {
	fn from(v: Arc<Vec<u8>>) -> Self {
		Self(v)
	}
}

impl From<Vec<u8>> for Msg {
	fn from(v: Vec<u8>) -> Self {
		Self::new(v)
	}
}

impl Into<Arc<Vec<u8>>> for Msg {
	fn into(self) -> Arc<Vec<u8>> {
		self.0
	}
}

// Buffering update dispatcher singleton.
//
// Never access Pulsar from the Registry, as Pulsar accesses it. Can result in
// deadlocks.
#[derive(Default)]
struct Pulsar {
	// Active feeds
	feeds: HashMap<u64, Feed>,

	// Global feed instance
	global: FeedCommon,
}

impl Pulsar {
	// Initialize with feed data as JSON
	fn init(&mut self, feed_data: &[u8]) -> serde_json::Result<()> {
		self.feeds =
			serde_json::from_slice::<HashMap<u64, FeedData>>(feed_data)?
				.into_iter()
				.map(|(thread, data)| (thread, Feed::new(data)))
				.collect();
		Ok(())
	}

	// Register a new thread and allocate its resources
	fn create_thread(&mut self, thread: u64) {
		self.global.clear_cache();

		let mut f = Feed::new(FeedData {
			thread: thread,
			..Default::default()
		});
		f.insert_post(thread);
		self.feeds.insert(thread, f);

		todo!("pass message to global feed")
	}

	// Deallocate thread resources and redirect all of its clients
	fn remove_thread(&mut self, id: u64) {
		self.global.clear_cache();

		todo!(concat!(
			"Remove feed data, redirect clients on thread deletion and ",
			" pass message to global feed"
		))
	}

	// Log an item has not been found
	fn log_not_found(label: &str, id: impl std::fmt::Debug) {
		bindings::log_error(&format!(
			"{} not found: {:?}\n{:?}",
			label,
			id,
			backtrace::Backtrace::new()
		))
	}

	fn mod_thread(&mut self, thread: u64, handler: impl FnOnce(&mut Feed)) {
		match self.feeds.get_mut(&thread) {
			Some(f) => {
				self.global.clear_cache();
				f.clear_cache();
				handler(f);
			}
			None => Self::log_not_found("thread", thread),
		}
	}

	// Insert an image into an allocated post
	fn insert_image(&mut self, req: ImageInsertionReq) {
		self.mod_thread(req.thread, |f| {
			match f.data.open_posts.get_mut(&req.post) {
				Some(p) => {
					p.has_image = true;
					p.spoilered_image = req.img.common.spoilered;
					f.encode_post_message(
						req.post,
						MessageType::InsertImage,
						&InsertImage {
							post: req.post,
							image: req.img,
						},
					);
				}
				None => {
					Self::log_not_found("open post", (req.thread, req.post))
				}
			}
		})
	}

	// Clean up expired recent posts
	fn clean_up(&mut self) {
		let threshold = (SystemTime::now() - Duration::from_secs(60 * 15))
			.elapsed()
			.unwrap_or(Duration::from_secs(0))
			.as_secs();
		self.feeds.par_iter_mut().for_each(|(_, feed)| {
			feed.data
				.recent_posts
				.retain(|_, created_on| *created_on > threshold)
		})
	}

	fn send_messages(&mut self) {
		// TODO: Make client filter recent posts by creation timestamp to the
		// last 15 min

		// Need a snapshot of the required registry fields for atomicity
		let (all_clients, clients_by_feed) = registry::snapshot_threads(|sm| {
			let mut not_ready = Vec::<(u64, HashSet<u64>)>::new();
			for (feed, clients) in sm.0.drain() {
				if feed == 0 {
					self.global.need_init.extend(clients);
					continue;
				}
				match self.feeds.get_mut(&feed) {
					Some(f) => f.common.need_init.extend(clients),
					None => not_ready.push((feed, clients)),
				};
			}
			if not_ready.len() != 0 {
				sm.0 = not_ready.into_iter().collect();
			}
		});

		// Used for aggregation of messages in parallel
		#[derive(Default)]
		struct MessageSet {
			// Each aggregated into one message for I/O efficiency after the
			// main filter_map(). Doing it inside would create too much
			// nesting and require more reallocations.
			global_init_parts: Vec<Msg>,
			global_feed_messages: Vec<Vec<u8>>,

			// Aggregated into one message for I/O efficiency inside the main
			// filter_map(), as most of the time, global messages will not be
			// concatenated with them.
			thread_messages: HashMap<u64, Msg>,
		}

		let build_global_init = !self.global.need_init.is_empty()
			&& self.global.init_msg_cache.is_none();

		// Aggregate feed messages to send
		let mut messages: MessageSet = self
			.feeds
			.par_iter_mut()
			.filter_map(|(id, f)| -> Option<MessageSet> {
				if !build_global_init && f.common.need_init.is_empty() {
					return None;
				}

				#[rustfmt::skip]
				macro_rules! try_encode {
					($result:expr) => {{
						match $result {
							Ok(v) => v,
							Err(err) => {
								f.common.log_encode_error(err);
								return None;
							}
						}
					}};
				}

				let thread_messages: HashMap<u64, Msg> = match (
					f.common.need_init.len() != 0,
					f.common.pending.take(),
					clients_by_feed.get(id),
				) {
					(true, None, Some(clients)) => {
						let msg = try_encode!(f.get_init_msg());
						f.common
							.need_init
							.drain()
							.filter(|c| clients.contains(&c))
							.map(|c| (c, msg.clone()))
							.collect()
					}
					(true, Some(pending), Some(clients)) => {
						let msg = try_encode!(pending.finish());
						// Init messages should be sent first to maintain
						// event sequentiality
						let with_init = Msg::new(Encoder::join(&[
							try_encode!(f.get_init_msg()).as_ref(),
							msg.as_slice(),
						]));
						let single = Msg::new(msg);

						clients
							.iter()
							.map(|c| {
								(
									*c,
									if f.common.need_init.contains(c) {
										with_init.clone()
									} else {
										single.clone()
									},
								)
							})
							.collect()
					}
					(false, Some(pending), Some(clients)) => {
						let msg = Msg::new(try_encode!(pending.finish()));
						clients
							.iter()
							.cloned()
							.map(|c| (c, msg.clone()))
							.collect()
					}
					// If no clients, simply drop the full encoder
					_ => Default::default(),
				};
				// Always clear clients needing init, as they were either
				// handled above or ignored due to navigating away or
				// disconnecting
				f.common.need_init.clear();

				// TODO: Compute splice messages from stored post body
				// string pairs before regenerating init messages.
				// TODO: Parallelize not only on threads, but also on splice
				// pairs

				Some(MessageSet {
					thread_messages: thread_messages,
					global_init_parts: if build_global_init {
						vec![try_encode!(f.get_global_init_msg_part())]
					} else {
						Default::default()
					},
					global_feed_messages: match f.pending_global.take() {
						Some(pending) => vec![try_encode!(pending.finish())],
						None => Default::default(),
					},
				})
			})
			.reduce(
				|| Default::default(),
				|mut a, b| {
					a.global_init_parts.extend(b.global_init_parts);
					a.global_feed_messages.extend(b.global_feed_messages);
					a.thread_messages.extend(b.thread_messages);
					a
				},
			);

		let mut messages_by_client = messages.thread_messages;

		// Assign global feed messages to clients
		match (
			!messages.global_init_parts.is_empty(),
			!messages.global_feed_messages.is_empty(),
			clients_by_feed.get(&0),
		) {
			(true, false, Some(clients)) => {
				let msg = Msg::new(Encoder::join(messages.global_init_parts));
				for c in self
					.global
					.need_init
					.drain()
					.filter(|c| clients.contains(&c))
				{
					messages_by_client.insert(c, msg.clone());
				}
			}
			(true, true, Some(clients)) => {
				let single =
					Msg::new(Encoder::join(messages.global_feed_messages));
				// Init messages should be sent first to maintain
				// event sequentiality
				messages.global_init_parts.push(single.clone());
				let with_init =
					Msg::new(Encoder::join(messages.global_init_parts));

				for c in clients.iter().cloned() {
					messages_by_client.insert(
						c,
						if self.global.need_init.contains(&c) {
							with_init.clone()
						} else {
							single.clone()
						},
					);
				}
			}
			(false, true, Some(clients)) => {
				let msg =
					Msg::new(Encoder::join(messages.global_feed_messages));
				for c in clients.iter().cloned() {
					messages_by_client.insert(c, msg.clone());
				}
			}
			_ => (),
		}
		// Always clear clients needing init, as they were either handled above
		// or ignored due to navigating away or disconnecting
		self.global.need_init.clear();

		// Merge server-wide messages to all clients.
		// Not very efficient, but that is fine. These happen rarely.
		if let Some(pending) = self.global.pending.take() {
			match pending.finish() {
				Ok(buf) => {
					messages_by_client.par_iter_mut().for_each(
						|(_, queued)| {
							*queued = Msg::new(Encoder::join(&[
								queued.as_ref(),
								buf.as_ref(),
							]));
						},
					);
					let msg = Msg::new(buf);
					for c in all_clients
						.iter()
						.filter(|c| !messages_by_client.contains_key(&c))
						.cloned()
						.collect::<Vec<_>>()
					{
						messages_by_client.insert(c, msg.clone());
					}
				}
				Err(err) => self.global.log_encode_error(err),
			};
		}

		// Send all messages in parallel to maximize parallelism of the Go side
		messages_by_client
			.into_par_iter()
			.for_each(|(client, msg)| {
				bindings::write_message(client, msg.into());
			})
	}
}

pub struct ImageInsertionReq {
	thread: u64,
	post: u64,
	img: Image,
}

// Request to pulsar
pub enum Request {
	// Initialize a freshly-created thread
	CreateThread(u64),

	// Deallocate thread resources and redirect all of its clients
	RemoveThread(u64),

	// Insert an image into an allocated post
	InsertImage(ImageInsertionReq),
}

// Alias Result for sending a request to Pulsar
pub type SendResult = Result<(), SendError<Request>>;

fn send_request(req: Request) -> SendResult {
	unsafe { REQUEST.as_ref().unwrap().lock().unwrap().clone() }.send(req)
}

// Initialize a freshly-created thread
pub fn create_thread(id: u64) -> SendResult {
	send_request(Request::CreateThread(id))
}

// Deallocate thread resources and redirect all of its clients
pub fn remove_thread(id: u64) -> SendResult {
	send_request(Request::RemoveThread(id))
}

// Insert an image into an allocated post
pub fn insert_image(thread: u64, post: u64, img: Image) -> SendResult {
	send_request(Request::InsertImage(ImageInsertionReq {
		thread: thread,
		post: post,
		img: img,
	}))
}
