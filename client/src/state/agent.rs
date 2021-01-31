use super::{write, FeedID, Focus, Location, Post, Thread};
use crate::{connection::send, util};
use common::{
	debug_log,
	payloads::{ThreadCreationNotice, ThreadWithPosts},
	util::DoubleSetMap,
	MessageType,
};
use indexmap::IndexSet;
use serde::{Deserialize, Serialize};
use std::collections::{hash_map::Entry, HashMap};
use wasm_bindgen::JsCast;
use yew::{
	agent::{AgentLink, Bridge, Context, Dispatched, HandlerId},
	services::render::{RenderService, RenderTask},
	Callback, Component, ComponentLink,
};

// TODO: resync on disconnect
// TODO: received page trigger
// TODO: request new pages to be fetched on current thread

// TODO: differentiate between updates coming from the index and thread feeds
// to prevent duplicate messages. This can be done via a boolean on all post
// update messages (implement a trait that sets a boolean for these messages to
// be called inside write_post_message).

// TODO: there is technically a data race, if the sure rapidly switches between
// 2 feeds. Need to figure a way to address it.

/// Location setting flags
const PUSH_STATE: u8 = 1;
const SET_STATE: u8 = 1 << 1;
const NO_TRIGGER: u8 = 1 << 3;

/// Subscribe to updates of a value type
pub enum Request {
	NotifyChange(Vec<Change>),

	/// Change the current notifications a client is subscribed to
	ChangeNotifications {
		remove: Vec<Change>,
		add: Vec<Change>,
	},

	/// Fetch feed data
	FetchFeed(Location),

	/// Navigate to the app to a different feed
	NavigateTo {
		loc: Location,
		flags: u8,
	},

	/// Set or delete the ID of the currently used KeyPair
	SetKeyID(Option<uuid::Uuid>),

	/// Insert a new thread into the registry
	InsertThread(ThreadCreationNotice),

	/// Set post as created by this user
	SetMine(u64),

	/// Set ID of currently open post
	SetOpenPostID(Option<u64>),

	/// Register a page's posts in the application state
	RegisterPage(Vec<Post>),

	/// Register thread metainformation
	RegisterThread(Thread),

	/// Register threads passed from the thread index feed
	RegisterThreads(Vec<ThreadWithPosts>),

	/// Set tags used on threads
	SetUsedTags(Vec<String>),

	/// Set time correction between the server and client
	SetTimeCorrection(i32),
}

/// Selective changes of global state to be notified on
#[derive(Serialize, Deserialize, Eq, PartialEq, Hash, Copy, Clone, Debug)]
pub enum Change {
	/// Change of location the app is navigated to
	Location,

	/// Authentication key pair has been set by user
	KeyPair,

	/// Change to any field of Options
	Options,

	/// Change to any field of the Configs
	Configs,

	/// Change in tags used on threads
	UsedTags,

	/// Subscribe to changes of the list of threads
	ThreadList,

	/// Subscribe to thread data changes, excluding the post content level.
	/// This includes changes to the post set of threads.
	Thread(u64),

	/// Subscribe to any changes to a post
	Post(u64),

	/// Change in time correction value
	TimeCorrection,
}

/// Abstraction over AgentLink and ComponentLink
pub trait Link {
	type Message;

	fn make_callback<F>(&self, f: F) -> Callback<()>
	where
		F: Fn(()) -> Self::Message + 'static;
}

impl<A: yew::agent::Agent> Link for AgentLink<A> {
	type Message = A::Message;

	fn make_callback<F>(&self, f: F) -> Callback<()>
	where
		F: Fn(()) -> Self::Message + 'static,
	{
		self.callback(f)
	}
}

impl<C: Component> Link for ComponentLink<C> {
	type Message = C::Message;

	fn make_callback<F>(&self, f: F) -> Callback<()>
	where
		F: Fn(()) -> Self::Message + 'static,
	{
		self.callback(f)
	}
}

/// Helper for storing a hook into state updates in the client struct
pub struct HookBridge {
	#[allow(unused)]
	bridge: Box<dyn Bridge<Agent>>,
}

impl HookBridge {
	pub fn send(&mut self, req: Request) {
		self.bridge.send(req);
	}
}

/// Crate hooks into state changes
pub fn hook<L, F>(link: &L, changes: Vec<Change>, f: F) -> HookBridge
where
	L: Link,
	F: Fn(()) -> L::Message + 'static,
{
	use yew::agent::Bridged;

	let mut b = HookBridge {
		bridge: Agent::bridge(link.make_callback(f)),
	};
	if !changes.is_empty() {
		b.bridge.send(Request::NotifyChange(changes));
	}
	b
}

pub enum Message {
	Focus(Focus),
	PoppedState,
}

/// Feed synchronization state
#[derive(Debug)]
enum FeedSyncState {
	/// No feed requested yet
	NotRequested,

	/// Feed requested but not all requested data received
	Receiving {
		/// Location being synced to
		loc: Location,

		/// Thread metainformation received from the server
		thread: Option<Thread>,

		/// Thread pages that need to be received or are already
		pages: HashMap<i32, Option<Vec<Post>>>,

		/// Flags passed during the fetch
		flags: u8,
	},

	/// Fully synced to server feed
	Synced {
		/// Feed ID
		feed: FeedID,

		/// Thread pages that need to be received or are already.
		/// `true` means it has been received.
		pages: HashMap<u32, bool>,
	},
}

// /// Arguments used for merging a feed from websocket and JSON API data
// struct FeedMergeArgs {
// 	loc: Location,
// 	flags: u8,
// 	from_json: Vec<ThreadWithPosts>,
// 	from_websocket: HashMap<u64, FeedData>,
// }

/// Global state storage and propagation agent
pub struct Agent {
	link: AgentLink<Self>,

	/// Clients hooked into change notifications
	hooks: DoubleSetMap<Change, HandlerId>,

	/// Change notifications pending flushing to clients.
	queued_triggers: IndexSet<HandlerId>,

	/// Task used to defer actions to the next animation frame
	render_task: Option<RenderTask>,

	/// State of synchronization to the current or pending feed
	feed_sync_state: FeedSyncState,
}

impl yew::agent::Agent for Agent {
	type Reach = Context<Self>;
	type Message = Message;
	type Input = Request;
	type Output = ();

	fn create(link: AgentLink<Self>) -> Self {
		util::add_static_listener(
			util::window(),
			"popstate",
			true,
			link.callback(|_: web_sys::Event| Message::PoppedState),
		);

		Self {
			link,
			hooks: DoubleSetMap::default(),
			render_task: None,
			feed_sync_state: FeedSyncState::NotRequested,
			queued_triggers: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) {
		use Message::*;

		match msg {
			Focus(f) => {
				use self::Focus::*;
				use util::document;
				use web_sys::HtmlElement;

				fn banner_height() -> f64 {
					document()
						.get_element_by_id("banner")
						.map(|el| {
							el.dyn_into::<HtmlElement>()
								.ok()
								.map(|el| el.offset_height() - 5)
						})
						.flatten()
						.unwrap_or_default() as f64
				}

				util::window().scroll_with_x_and_y(
					0.0,
					match f {
						Top => banner_height(),
						Bottom => document()
							.document_element()
							.map(|el| el.scroll_height())
							.unwrap_or_default() as f64,
						Post(id) => document()
							.get_element_by_id(&format!("p-{}", id))
							.map(|el| {
								el.dyn_into::<HtmlElement>().ok().map(|el| {
									el.offset_height() as f64 + banner_height()
								})
							})
							.flatten()
							.unwrap_or_default(),
					},
				);
			}
			PoppedState => self.set_location(Location::from_path(), SET_STATE),
		}

		self.flush_triggers();
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		use Request::*;

		match req {
			NotifyChange(h) => {
				for h in h {
					self.hooks.insert(h, id);
				}
			}
			ChangeNotifications { remove, add } => {
				for h in remove {
					self.hooks.remove_by_key_value(&h, &id);
				}
				for h in add {
					self.hooks.insert(h, id);
				}
			}
			NavigateTo { loc, flags } => self.set_location(loc, flags),
			FetchFeed(loc) => {
				self.try_sync_feed(&loc, 0);
			}
			SetKeyID(id) => util::with_logging(|| {
				write(|s| {
					s.key_pair.id = id;
					s.key_pair.store()?;
					Ok(())
				})
			}),
			InsertThread(n) => {
				write(|s| {
					s.threads.insert(
						n.id,
						Thread {
							id: n.id,
							page_count: 1,
							subject: n.subject,
							tags: n.tags,
							bumped_on: n.time,
							created_on: n.time,
							post_count: 1,
							image_count: 0,
						},
					);
					s.register_post(Post {
						id: n.id,
						thread: n.id,
						page: 0,

						created_on: n.time,
						open: true,

						// TODO: set this from modal
						opts: Default::default(),

						body: Default::default(),
						image: Default::default(),
					});
					self.trigger(&Change::ThreadList);
					self.trigger(&Change::Thread(n.id));
					self.trigger(&Change::Post(n.id));
				});
			}
			RegisterPage(posts) => self.register_page(posts),
			RegisterThreads(threads) => self.register_threads(threads),
			RegisterThread(thread) => match &mut self.feed_sync_state {
				FeedSyncState::Receiving {
					loc, thread: dst, ..
				} if loc.feed.as_u64() == thread.id => {
					*dst = Some(thread);
				}
				_ => (),
			},
			SetMine(id) => {
				// TODO: persist to DB
				write(|s| s.mine.insert(id));
			}
			SetOpenPostID(id) => {
				write(|s| {
					s.open_post_id = id;
					if let Some(id) = s.open_post_id {
						use crate::post::posting;

						posting::Agent::dispatcher()
							.send(posting::Request::SetAllocated(id));
						if let Some(affected) =
							self.hooks.get_by_key(&Change::Post(id))
						{
							for h in affected {
								self.link.respond(*h, ());
							}
						}
					}
				});
			}
			SetUsedTags(tags) => {
				write(|s| {
					s.used_tags = tags.into();
				});
				self.trigger(&Change::UsedTags);
			}
			SetTimeCorrection(c) => {
				write(|s| {
					s.time_correction = c;
				});
				self.trigger(&Change::TimeCorrection);
			}
		};

		self.flush_triggers();
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.hooks.remove_by_value(&id);
	}
}

impl Agent {
	/// Schedule to send change notification to hooked clients.
	///
	/// Triggers need to be flushed to send the notifications.
	///
	/// Trigger these updates in hierarchical order to make any upper level
	/// components switch rendering modes and not cause needless updates
	/// on deleted child components.
	///
	/// Notifications are buffered to reduce double notification chances and any
	/// overhead of double sending and double handling.
	fn trigger(&mut self, h: &Change) {
		if let Some(subs) = self.hooks.get_by_key(h) {
			for id in subs.iter() {
				self.queued_triggers.insert(*id);
			}
		}
	}

	/// Flush queued notifications to clients
	fn flush_triggers(&mut self) {
		for id in self.queued_triggers.drain(0..) {
			self.link.respond(id, ());
		}
	}

	/// Set app location and propagate changes
	fn set_location(&mut self, new: Location, flags: u8) {
		write(|s| {
			let old = s.location.clone();
			if old == new {
				return;
			}

			debug_log!(
				"set_location",
				format!("{:?} -> {:?}, flags={}", s.location, new, flags)
			);

			let mut try_to_sync = true;

			// Check, if feed did not change, only requesting a new page
			match (&mut self.feed_sync_state, &new.feed) {
				(
					FeedSyncState::Synced { feed, pages },
					FeedID::Thread { id, page: new_page },
				) if &feed.as_u64() == id => {
					let mut new_page = *new_page;
					if new_page < -1 {
						util::log_and_alert_error(
							&"requested negative page ID smaller than -1",
						);
						return;
					}
					if new_page < 0 {
						new_page += s.get_synced_thread(id).page_count as i32;
					}
					if let Entry::Vacant(e) = pages.entry(new_page as u32) {
						e.insert(false);
						send(MessageType::Page, &new_page);
						try_to_sync = false;
					}
				}
				_ => (),
			};

			if try_to_sync && self.try_sync_feed(&new, flags) {
				return;
			}

			self.set_location_no_sync(s, new, flags);
		});
	}

	/// Set app location and propagate changes without trying to sync the feed
	/// first, if needed
	fn set_location_no_sync(
		&mut self,
		s: &mut super::State,
		new: Location,
		flags: u8,
	) {
		if flags & SET_STATE != 0 {
			s.location = new.clone();
			if flags & NO_TRIGGER == 0 {
				self.trigger(&Change::Location);
			}
			if let Some(f) = new.focus.clone() {
				self.render_task = RenderService::request_animation_frame(
					self.link.callback(move |_| Message::Focus(f.clone())),
				)
				.into();
			}
		}

		if flags & PUSH_STATE != 0 {
			// TODO: Set last scroll position on back and hash navigation
			// using replace_state()
			util::with_logging(|| {
				util::window().history()?.push_state_with_url(
					&wasm_bindgen::JsValue::NULL,
					"",
					Some(&new.path()),
				)?;
				Ok(())
			});
		}
	}

	/// Register posts of a page in the application state
	fn register_page(&mut self, posts: Vec<Post>) {
		use FeedSyncState::*;

		let (thread_id, page) = match posts.first() {
			Some(p) => (p.thread, p.page),
			None => return,
		};

		// Import threads only once we know the import is valid, to not
		// overwrite data from different feeds

		match &mut self.feed_sync_state {
			Receiving {
				loc,
				pages,
				flags,
				thread,
			} if thread.is_some() && loc.feed.as_u64() == thread_id => {
				pages.insert(page as i32, Some(posts));
				// Also insert the page number counted from the back to prevent
				// duplicate requests
				pages.insert(
					thread.as_ref().unwrap().page_count as i32 - page as i32,
					None,
				);

				if pages
					.iter()
					.filter(|(id, _)| **id >= 0)
					.all(|(_, posts)| posts.is_some())
				{
					use std::mem::take;

					let pages = take(pages);
					let loc = take(loc);
					let thread = thread.take().unwrap();
					let flags = *flags;
					self.feed_sync_state = Synced {
						feed: loc.feed.clone(),
						pages: pages
							.keys()
							.map(|id| (*id as u32, true))
							.collect(),
					};

					write(|s| {
						self.trigger(&Change::ThreadList);
						self.trigger(&Change::Thread(thread_id));
						s.threads.insert(thread.id, thread);

						for p in pages
							.into_iter()
							.filter(|(id, _)| *id >= 0)
							.map(|(_, p)| p.unwrap().into_iter())
							.flatten()
						{
							self.trigger(&Change::Post(p.id));
							s.register_post(p);
						}

						self.set_location_no_sync(s, loc, flags);
					});
				}
			}
			Synced { feed, pages } if feed.as_u64() == thread_id => {
				pages.insert(page, true);

				self.trigger(&Change::Thread(thread_id));
				write(|s| {
					for p in posts {
						self.trigger(&Change::Post(p.id));
						s.register_post(p);
					}
				});
			}
			_ => (),
		};
	}

	/// Register threads passed from the thread index feed
	fn register_threads(&mut self, threads: Vec<ThreadWithPosts>) {
		match &mut self.feed_sync_state {
			FeedSyncState::Receiving { loc, flags, .. }
				if loc.feed.as_u64() == 0 =>
			{
				let loc = std::mem::take(loc);
				let flags = *flags;
				self.feed_sync_state = FeedSyncState::Synced {
					feed: loc.feed.clone(),
					pages: Default::default(),
				};

				write(|s| {
					self.trigger(&Change::ThreadList);
					for t in threads {
						self.trigger(&Change::Thread(t.thread_data.id));
						s.threads.insert(t.thread_data.id, t.thread_data);

						for (_, p) in t.posts {
							self.trigger(&Change::Post(p.id));
							s.register_post(p);
						}
					}

					self.set_location_no_sync(s, loc, flags);
				});
			}
			_ => (),
		};
	}

	/// Fetch feed data from server, if needed.
	/// Returns, if a fetch is currently in progress.
	fn try_sync_feed(&mut self, new: &Location, flags: u8) -> bool {
		use crate::connection::{Connection, Request};
		use common::Encoder;

		let new_feed_num = new.feed.as_u64();

		// Clear any previous feed sync state, if feed changed
		match &mut self.feed_sync_state {
			// Already receiving data
			FeedSyncState::Receiving { loc, pages, .. }
				if loc.feed.as_u64() == new_feed_num =>
			{
				// Propagate non-feed updates
				*loc = new.clone();

				match &new.feed {
					FeedID::Thread { page, .. } => {
						if let Entry::Vacant(e) = pages.entry(*page) {
							// Requested another page
							e.insert(None);
							send(MessageType::Page, page);
						}
					}
					_ => (),
				};

				true
			}
			// If feed did not change, this is a page navigation within the
			// same feed. Keep the init data as there won't be any new received.
			FeedSyncState::Synced { feed, .. }
				if feed.as_u64() == new_feed_num =>
			{
				false
			}
			_ => {
				debug_log!("fetching");
				util::with_logging(|| {
					let mut e = Encoder::default();
					let mut pages = HashMap::new();

					e.write_message(MessageType::Synchronize, &new_feed_num)?;

					match &new.feed {
						FeedID::Thread { page, .. } => {
							e.write_message(MessageType::Page, page)?;
							pages.insert(*page, None);
						}
						_ => (),
					};

					Connection::dispatcher().send(Request::Send(e.finish()?));
					self.feed_sync_state = FeedSyncState::Receiving {
						loc: new.clone(),
						flags,
						pages,
						thread: None,
					};
					Ok(true)
				})
			}
		}
	}
}

/// Navigate to the app to a different location
pub fn navigate_to(loc: Location) {
	Agent::dispatcher().send(Request::NavigateTo {
		loc,
		flags: PUSH_STATE | SET_STATE,
	});
}
