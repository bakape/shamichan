use super::common::{DynResult, SetMap};
use super::{bindings, registry};
use protocol::*;
use rayon::prelude::*;
use std::cell::RefCell;
use std::collections::{HashMap, HashSet};
use std::sync::{mpsc, Arc, Mutex, Once};

// TODO: Optimise global feed with partial init message caching
// TODO: Remove feed data on thread deletion
// TODO: Ensure client does not try to mutate feed data before it has been
// initialized Those NOP on the server.

static ONCE: Once = Once::new();

// For sending feed init data to Pulsar. Clone to use.
static mut SEND_FEED_DATA: Option<
	Mutex<mpsc::Sender<(u64, Result<FeedData, String>)>>,
> = None;

// Init module state
pub fn init() {
	ONCE.call_once(|| {
		let (s, r) = mpsc::channel();
		unsafe { SEND_FEED_DATA = Some(Mutex::new(s)) };
		let mut p = Pulsar {
			feeds: Default::default(),
			scheduled_feed_inits: Default::default(),
			need_init: Default::default(),
			receive_init_data: r,
		};
		std::thread::spawn(move || {
			let t = std::time::Duration::from_millis(100);
			loop {
				// Block until messages are sent to the Go side to guarantee
				// sequentiality
				p.send_messages();
				std::thread::sleep(t); // At end for faster server startup
			}
		});
	})
}

// Update feed. Either a thread feed or the global thread index feed.
#[derive(Default, Debug)]
struct Feed {
	// Cached encoded initialization message buffer
	init_msg_cache: Option<Arc<Vec<u8>>>,

	// Pending message streaming encoder
	pending: Encoder<Vec<u8>>,

	// Current active feed data
	data: FeedData,
}

impl Feed {
	// Encode and cache feed init message
	fn gen_init_msg(&mut self) -> DynResult<Arc<Vec<u8>>> {
		let msg = Arc::new({
			let mut enc = Encoder::new(Vec::new());
			enc.write_message(MessageType::FeedInit, &self.data)?;
			enc.finish()?
		});
		self.init_msg_cache = Some(msg.clone());
		Ok(msg)
	}
}

// Buffering update dispatcher singleton.
//
// Never access Pulsar from the Registry, as Pulsar accesses it. Can result in
// deadlocks.
struct Pulsar {
	// Active feeds.
	//
	// Uses 2 tier locking behind a RefCell to allow concurrent writes to
	// different feeds.
	feeds: HashMap<u64, Mutex<RefCell<Feed>>>,

	// Receive initial feed data from Go
	receive_init_data: mpsc::Receiver<(u64, Result<FeedData, String>)>,

	// Feeds currently retrieving init data
	scheduled_feed_inits: HashSet<u64>,

	// Have not yet had their feed initialization messages sent.
	// Mapped by thread.
	//
	// Need to wrap in refcell to avoid internal mutability issues.
	need_init: RefCell<SetMap<u64, u64>>,
}

impl Pulsar {
	fn send_messages(&mut self) {
		// Consume any finished feed data fetches
		for (id, res) in self.receive_init_data.try_iter() {
			self.scheduled_feed_inits.remove(&id);
			match res {
				Err(e) => bindings::log_error(&format!(
					"could not init feed data: id={} err={}",
					id, e
				)),
				Ok(mut data) => {
					data.feed = id; // Ensure is set before encoding
					self.feeds.insert(
						id,
						Mutex::new(RefCell::new(Feed {
							data: data,
							..Default::default()
						})),
					);
				}
			}
		}

		// Need a snapshot of the required registry fields for atomicity
		let mut by_thread = SetMap::new();
		registry::snapshot_threads(
			&mut by_thread,
			&mut *self.need_init.borrow_mut(),
		);

		let initialized = {
			let need_init = &*self.need_init.borrow();

			// Schedule any needed feed data fetches
			for feed in need_init
				.keys()
				.filter(|x| {
					!self.feeds.contains_key(x)
						&& !self.scheduled_feed_inits.contains(x)
				})
				.collect::<Vec<_>>()
			{
				self.scheduled_feed_inits.insert(*feed);
				bindings::get_feed_data(*feed);
			}

			// Generate and send feed to client.
			//
			// Return feeds with successfully sent init messages.
			self.feeds
				.par_iter_mut()
				.filter_map(|(id, fl)| {
					let fm = fl.lock().unwrap(); // Hold lock guard
					let mut f = fm.borrow_mut();

					// TODO: Compute splice messages from stored post body
					// string pairs before regenerating init messages.
					// TODO: Parallelize not only on threads, but also on splice
					// pairs

					macro_rules! log_error {
						// This should never happen, but log it and halt
						// execution, if it does
						($msg:literal, $err:expr) => {
							bindings::log_error(&format!(
								concat!($msg, ": id={} err={:?}"),
								id, $err
								));
							return None;
						};
					}

					// Generate feed init messages for clients needing them.
					let re = match need_init.get(&id) {
						None => None,
						Some(clients) => {
							let msg: Arc<Vec<u8>> = match &mut f.init_msg_cache
							{
								Some(msg) => msg.clone(),
								None => match f.gen_init_msg() {
									Ok(msg) => msg,
									Err(e) => {
										log_error!(
											"could not encode feed data",
											e
										);
									}
								},
							};

							// Send feed init messages first to maintain
							// sequentiality
							for cl in clients {
								bindings::write_message(*cl, msg.clone());
							}

							Some(*id)
						}
					};

					// Send all pending messages to subscribed clients
					if !f.pending.empty() {
						let buf = match f.pending.reset(Vec::new()) {
							Ok(buf) => buf,
							Err(e) => {
								log_error!("could not reset encoder", e);
							}
						};
						// If no clients, simply drop the buffer
						if let Some(clients) = by_thread.get(id) {
							let msg = Arc::new(buf);
							for cl in clients {
								bindings::write_message(*cl, msg.clone());
							}
						}
					}

					re
				})
				.collect::<Vec<u64>>()
		};
		let mut need_init = self.need_init.borrow_mut();
		for id in initialized {
			need_init.remove_key(&id);
		}
	}
}

// Receive previously requested thread data encoded as JSON
pub fn receive_feed_data(id: u64, buf: Result<&[u8], String>) {
	init();

	{
		// Ensure early mutex guard drop with a separate scope
		unsafe { SEND_FEED_DATA.as_ref() }
			.unwrap()
			.lock()
			.unwrap()
			.clone()
	}
	.send((
		id,
		match buf {
			Err(e) => Err(e),
			Ok(buf) => serde_json::from_slice(buf).map_err(|e| e.to_string()),
		},
	))
	.expect("pulsar receiver not reachable");
}
