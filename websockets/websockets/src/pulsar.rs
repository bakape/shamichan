use super::common::SetMap;
use super::{bindings, registry};
use protocol::*;
use rayon::prelude::*;
use std::collections::VecDeque;
use std::collections::{hash_map, HashMap, HashSet};
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
pub fn init(feed_data: &[u8]) -> serde_json::Result<()> {
	let (sdr, recv) = channel();
	unsafe {
		REQUEST = Some(Mutex::new(sdr));
	}
	let mut p: Pulsar = Default::default();
	p.init(feed_data)?;
	std::thread::spawn(move || {
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
			let dur = Duration::from_millis(10) - (Instant::now() - started);
			if dur.as_millis() > 0 {
				std::thread::sleep(dur);
			}
		}
	});
	Ok(())
}

// Common to both thread feeds and the global Feed
#[derive(Default, Debug)]
struct FeedCommon {
	// Clients needing an init message sent
	need_init: HashSet<u64>,

	// Cached encoded initialization message buffer
	init_msg_cache: Option<Msg>,

	// Pending message streaming encoder
	pending: Option<Encoder>,
}

// Update feed. Either a thread feed or the global thread index feed.
#[derive(Default, Debug)]
struct Feed {
	// Thread of this feed
	thread: u64,

	common: FeedCommon,

	global_init_msg_part: Option<Msg>,

	// Pending messages for global thread index feed
	pending_global: Option<Encoder>,

	// Last 5 post IDs in thread
	last_5_posts: VecDeque<u64>,

	// Current active feed data.
	//
	// Options to account for clients arriving before a thread has been
	// initialized.
	data: Option<FeedData>,
}

impl Feed {
	// Create new wrapped Feed initialized with data
	fn new(data: FeedData) -> Self {
		Self {
			thread: data.thread,
			last_5_posts: Self::find_last_5(&data.recent_posts),
			data: Some(data),
			..Default::default()
		}
	}

	// Clear all cached values
	fn clear_cache(&mut self) {
		self.global_init_msg_part = None;
		self.common.init_msg_cache = None;
	}

	// Encode and cache feed init message or return cached one.
	//
	// Panics, if self.data is None.
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

	// Find last 5 posts added to thread
	fn find_last_5(recent_posts: &HashMap<u64, u64>) -> VecDeque<u64> {
		let mut last_5 = [0u64; 6];
		for id in recent_posts.keys() {
			last_5[5] = *id;
			last_5.sort_unstable_by(|a, b| a.partial_cmp(b).unwrap().reverse());
			last_5[5] = 0;
		}
		last_5[..last_5.iter().position(|x| *x == 0).unwrap_or(5)]
			.iter()
			.cloned()
			.collect()
	}

	// Return, if post should be included in global thread index
	fn include_in_global(&self, id: u64) -> bool {
		match self.data.as_ref() {
			Some(d) => {
				id == d.thread || self.last_5_posts.iter().any(|x| id == *x)
			}
			None => false,
		}
	}

	// Encode and cache global feed init message part or return cached one.
	//
	// Panics, if self.data is None.
	fn get_global_init_msg_part(&mut self) -> std::io::Result<Msg> {
		match &mut self.global_init_msg_part {
			Some(msg) => Ok(msg.clone()),
			None => {
				let msg = Msg::new({
					let mut enc = Encoder::new(Vec::new());

					enc.write_message(
						MessageType::FeedInit,
						&FeedData {
							thread: self.thread,

							// Only need post data from up to the last 5
							// posts and OP
							recent_posts: self
								.data
								.as_ref()
								.unwrap()
								.recent_posts
								.iter()
								.filter(|(id, _)| self.include_in_global(**id))
								.map(|(k, v)| (*k, *v))
								.collect(),
							open_posts: self
								.data
								.as_ref()
								.unwrap()
								.open_posts
								.iter()
								.filter(|(id, _)| self.include_in_global(**id))
								.map(|(k, v)| (*k, v.clone()))
								.collect(),
						},
					)?;

					enc.finish()?
				});
				self.global_init_msg_part = Some(msg.clone());
				Ok(msg)
			}
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
	fn create_thread(&mut self, data: FeedData) {
		// Account for clients arriving before a thread has been
		// initialized
		match self.feeds.entry(data.thread) {
			hash_map::Entry::Occupied(mut e) => {
				let mut r = e.get_mut();
				r.last_5_posts = Feed::find_last_5(&data.recent_posts);
				r.data = Some(data);
			}
			hash_map::Entry::Vacant(e) => {
				e.insert(Feed::new(data));
			}
		}
	}

	// Deallocate thread resources and redirect all of its clients
	fn remove_thread(&mut self, id: u64) {
		self.global.init_msg_cache = None;

		todo!("Remove feed data and redirect clients on thread deletion")
	}

	// Clean up expired recent posts
	fn clean_up(&mut self) {
		let threshold = (SystemTime::now() - Duration::from_secs(60 * 15))
			.elapsed()
			.unwrap()
			.as_secs();
		self.feeds.par_iter_mut().for_each(|(_, feed)| {
			if let Some(d) = &mut feed.data {
				d.recent_posts
					.retain(|_, created_on| *created_on > threshold)
			}
		})
	}

	// This should never happen, but log it and halt execution, if it does.
	// Caller should abort execution.
	fn log_encode_error<T: std::fmt::Display>(feed_id: T, err: std::io::Error) {
		bindings::log_error(&format!(
			"could not encode feed data: feed_id={} err={:?}",
			feed_id, err
		));
	}

	fn send_messages(&mut self) {
		// TODO: Make client filter recent posts by creation timestamp to the
		// last 15 min

		// Need a snapshot of the required registry fields for atomicity
		let (all_clients, clients_by_feed) =
			registry::snapshot_threads(|feed, clients| {
				if feed == 0 {
					self.global.need_init.extend(clients);
					return;
				}

				// Account for clients arriving before a thread has been
				// initialized
				match self.feeds.entry(feed) {
					hash_map::Entry::Occupied(mut e) => {
						e.get_mut().common.need_init.extend(clients);
					}
					hash_map::Entry::Vacant(e) => {
						e.insert(Feed {
							thread: feed,
							common: FeedCommon {
								need_init: clients,
								..Default::default()
							},
							..Default::default()
						});
					}
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
			// mixed concatenated
			thread_messages: HashMap<u64, Msg>,
		}

		let build_global_init = !self.global.need_init.is_empty()
			&& self.global.init_msg_cache.is_none();

		// Aggregate feed messages to send
		let mut messages: MessageSet = self
			.feeds
			.par_iter_mut()
			.filter_map(|(id, f)| -> Option<MessageSet> {
				if f.data.is_none()
					|| (!build_global_init && f.common.need_init.len() == 0)
				{
					return None;
				}

				#[rustfmt::skip]
				macro_rules! try_encode {
					($result:expr) => {{
						match $result {
							Ok(v) => v,
							Err(err) => {
								Self::log_encode_error(id, err);
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
			messages.global_init_parts.len() != 0,
			messages.global_feed_messages.len() != 0,
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
				Err(err) => Self::log_encode_error("global", err),
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

// Request to pulsar
pub enum Request {
	// Initialize a freshly-created thread
	CreateThread(FeedData),

	// Deallocate thread resources and redirect all of its clients
	RemoveThread(u64),
}

// Alias Result for sending a request to Pulsar
pub type SendResult = Result<(), SendError<Request>>;

fn send_request(req: Request) -> SendResult {
	unsafe { REQUEST.as_ref().unwrap().lock().unwrap().clone() }.send(req)
}

// Initialize a freshly-created thread
pub fn create_thread(data: FeedData) -> SendResult {
	send_request(Request::CreateThread(data))
}

// Deallocate thread resources and redirect all of its clients
pub fn remove_thread(id: u64) -> SendResult {
	send_request(Request::RemoveThread(id))
}
