use super::{write, FeedID, Focus, Location, Post, Thread};
use crate::util;
use common::{
	debug_log,
	payloads::{ThreadCreationNotice, ThreadWithPosts},
	util::DoubleSetMap,
};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use wasm_bindgen::JsCast;
use yew::{
	agent::{AgentLink, Bridge, Context, Dispatched, HandlerId},
	services::render::{RenderService, RenderTask},
	Callback, Component, ComponentLink,
};

/// Location setting flags
const PUSH_STATE: u8 = 1;
const SET_STATE: u8 = 1 << 1;
const FETCHED_JSON: u8 = 1 << 2;
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
	// TODO: port
	// /// Store feed initialization data for syncing to a feed
	// StoreFeedInitData {
	// 	feed: u64,
	// 	data: HashMap<u64, FeedData>,
	// },
	// /// Handle FeedSync synchronization routine result
	// SyncFeed {
	// 	loc: Location,
	// 	result: util::Result<(Vec<ThreadWithPosts>, Vec<Post>)>,
	// },
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

	/// Subscribe to changes of the list of threads
	ThreadList,

	/// Subscribe to thread data changes, excluding the post content level.
	/// This includes changes to the post set of threads.
	Thread(u64),

	/// Subscribe to any changes to a post
	Post(u64),
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
	// TODO: port
	// FetchedThreadIndex {
	// 	loc: Location,
	// 	data: Vec<ThreadWithPosts>,
	// 	flags: u8,
	// },
	// FetchedThread {
	// 	loc: Location,
	// 	data: ThreadWithPosts,
	// 	flags: u8,
	// },
	// FetchFailed(String),
	Focus(Focus),
	PoppedState,
}

// TODO: port
// /// Feed synchronization state
// #[derive(Debug)]
// enum FeedSync {
// 	/// No feed data received yet
// 	NoData,

// 	/// Only have the posts data from teh JSON API fetched
// 	JSONFetched {
// 		loc: Location,
// 		flags: u8,
// 		data: Vec<ThreadWithPosts>,
// 	},

// 	/// Only have the feed init data from the websocket
// 	WebsocketReceived {
// 		feed: u64,
// 		data: HashMap<u64, FeedData>,
// 		//
// 		// TODO: buffer post update messages till this resolves
// 	},

// 	/// Syncing discrepancies between the websocket feed init data data and
// 	/// API post JSON
// 	Syncing {
// 		loc: Location,
// 		flags: u8,
// 		threads: HashMap<u64, Thread>,
// 		posts: HashMap<u64, Post>,
// 		patches: HashMap<u64, FeedData>,
// 	},

// 	/// Fully synced to server feed
// 	Synced {
// 		feed: u64,
// 		/// Patches to apply on the downloaded JSON post data
// 		// TODO: keep updating this to keep it current in case of a same feed
// 		// page navigation
// 		patches: HashMap<u64, FeedData>,
// 	},
// }

// impl Default for FeedSync {
// 	fn default() -> Self {
// 		Self::NoData
// 	}
// }

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
	hooks: DoubleSetMap<Change, HandlerId>,
	fetch_task: Option<yew::services::fetch::FetchTask>,
	render_task: Option<RenderTask>,
	// TODO: port
	// feed_sync: FeedSync,
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
			fetch_task: None,
			render_task: None,
			// TODO: port
			// feed_sync: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) {
		use Message::*;

		match msg {
			// TODO: port
			// FetchedThreadIndex { loc, data, flags } => {
			// 	self.process_successful_feed_fetch(loc, data, flags);
			// }
			// FetchedThread { loc, data, flags } => {
			// 	self.process_successful_feed_fetch(loc, vec![data], flags);
			// }
			// FetchFailed(s) => {
			// 	util::log_and_alert_error(&s);
			// 	self.fetch_task = None;
			// }
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
				// Only called on app start, so pass FeedID::Unset
				// TODO: port
				// self.fetch_feed_data(loc, &Default::default(), 0);
			}
			// TODO: port
			// StoreFeedInitData { feed, data } => {
			// 	self.merge_feed_data(FeedSync::WebsocketReceived {
			// 		feed,
			// 		data,
			// 	});
			// }
			// SyncFeed { loc, result } => {
			// 	let (fetched_threads, fetched_posts) = match result {
			// 		Ok(t) => t,
			// 		Err(e) => {
			// 			util::log_and_alert_error(&e);
			// 			return;
			// 		}
			// 	};
			// 	let args = match &mut self.feed_sync {
			// 		FeedSync::Syncing {
			// 			loc: stored_loc,
			// 			flags,
			// 			threads,
			// 			posts,
			// 			patches,
			// 		} if stored_loc == &loc => {
			// 			use std::collections::hash_map::Entry::{
			// 				Occupied, Vacant,
			// 			};
			// 			use std::mem::{swap, take};

			// 			for t in fetched_threads {
			// 				if let Vacant(e) = threads.entry(t.thread_data.id) {
			// 					e.insert(t.thread_data);
			// 					for p in t.posts {
			// 						if let Vacant(e) = posts.entry(p.id) {
			// 							e.insert(p);
			// 						}
			// 					}
			// 				}
			// 			}
			// 			for mut p in fetched_posts {
			// 				match posts.entry(p.id) {
			// 					Vacant(e) => {
			// 						e.insert(p);
			// 					}
			// 					Occupied(mut e) => {
			// 						let ptr = e.get_mut();
			// 						if ptr.image.is_none() && p.image.is_some()
			// 						{
			// 							swap(&mut ptr.image, &mut p.image);
			// 						}
			// 					}
			// 				};
			// 			}

			// 			Some((
			// 				*flags,
			// 				take(threads),
			// 				take(posts),
			// 				take(patches),
			// 			))
			// 		}
			// 		_ => None,
			// 	};
			// 	if let Some((flags, threads, posts, patches)) = args {
			// 		self.complete_feed_sync(
			// 			loc, flags, threads, posts, patches,
			// 		);
			// 	}
			// }
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
							last_page: 0,
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
		};
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.hooks.remove_by_value(&id);
	}
}

impl Agent {
	/// Send change notification to hooked clients
	fn trigger(&self, h: &Change) {
		if let Some(subs) = self.hooks.get_by_key(h) {
			for id in subs.iter() {
				self.link.respond(*id, ());
			}
		}
	}

	/// Set app location and propagate changes
	fn set_location(&mut self, new: Location, flags: u8) {
		write(|s| {
			use super::FeedID::*;

			let old = s.location.clone();
			if old == new {
				return;
			}

			debug_log!(
				"set_location",
				format!("{:?} -> {:?}, flags={}", s.location, new, flags)
			);

			let need_fetch = flags & FETCHED_JSON == 0
				&& match (&old.feed, &new.feed) {
					(
						Thread {
							id: old_id,
							page: old_page,
						},
						Thread {
							id: new_id,
							page: new_page,
						},
					) => {
						new_id != old_id
							|| (old_page != new_page
						// Page number corrections do not need a refetch
							&& !(old_page == &-1 && new_page != &-1))
					}

					// Index/Catalog and Thread transitions always need a fetch
					(Thread { .. }, _) | (_, Thread { .. }) => true,

					// Catalog and Index transition do not need a fetch
					_ => false,
				};
			if need_fetch {
				debug_log!("fetching");
				// TODO: port
				// self.fetch_feed_data(new, &s.location, flags);
				return;
			}

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
		});
	}

	// TODO: port
	// /// Merge feed data fetched through the JSON API and websocket
	// fn merge_feed_data(&mut self, mut rhs: FeedSync) {
	// 	use std::mem::take;
	// 	use FeedSync::*;

	// 	let args = match (&mut self.feed_sync, &mut rhs) {
	// 		// Fresh connection
	// 		(NoData, JSONFetched { .. })
	// 		| (NoData, WebsocketReceived { .. }) => {
	// 			self.feed_sync = rhs;
	// 			None
	// 		}

	// 		// Syncing after feed change.
	// 		// If feed IDs do not match, it's a race condition and should be
	// 		// ignored.
	// 		(
	// 			JSONFetched {
	// 				loc,
	// 				flags,
	// 				data: from_json,
	// 			},
	// 			WebsocketReceived {
	// 				feed,
	// 				data: from_websocket,
	// 			},
	// 		)
	// 		| (
	// 			WebsocketReceived {
	// 				feed,
	// 				data: from_websocket,
	// 			},
	// 			JSONFetched {
	// 				loc,
	// 				flags,
	// 				data: from_json,
	// 			},
	// 		) if *feed == loc.feed.as_u64() => Some(FeedMergeArgs {
	// 			loc: take(loc),
	// 			flags: *flags,
	// 			from_json: take(from_json),
	// 			from_websocket: take(from_websocket),
	// 		}),

	// 		// Changed page without changing feed
	// 		(
	// 			Synced { feed, patches },
	// 			JSONFetched {
	// 				loc,
	// 				flags,
	// 				data: from_json,
	// 			},
	// 		) if *feed == loc.feed.as_u64() => Some(FeedMergeArgs {
	// 			loc: take(loc),
	// 			flags: *flags,
	// 			from_json: take(from_json),
	// 			from_websocket: take(patches),
	// 		}),

	// 		// Everything else is due to race conditions or programming errors
	// 		(dst @ _, src @ _) => {
	// 			util::log_warn(format!(
	// 				"failed to merge feed data: {:?} into {:?}",
	// 				dst, src
	// 			));
	// 			None
	// 		}
	// 	};
	// 	if let Some(args) = args {
	// 		self.merge_feed_sync(args);
	// 	}
	// }

	// TODO: port
	// /// Merge feed data from the JSON and websocket APIs, possibly fetching
	// /// missing data from the JSON API
	// fn merge_feed_sync(&mut self, args: FeedMergeArgs) {
	// 	use futures::Future;

	// 	let FeedMergeArgs {
	// 		loc,
	// 		flags,
	// 		from_json,
	// 		from_websocket,
	// 	} = args;

	// 	let mut threads =
	// 		HashMap::<u64, Thread>::with_capacity(from_json.len());
	// 	let mut posts = HashMap::<u64, Post>::new();
	// 	for t in from_json {
	// 		use std::collections::hash_map::Entry::*;

	// 		match threads.entry(t.thread_data.id) {
	// 			Occupied(mut e) => {
	// 				// Can contain multiple entries per thread due to fetching
	// 				// several pages
	// 				let ptr = e.get_mut();
	// 				ptr.page = std::cmp::max(ptr.page, t.thread_data.page);
	// 			}
	// 			Vacant(e) => {
	// 				e.insert(t.thread_data);
	// 			}
	// 		}
	// 		posts.extend(t.posts.into_iter().map(|p| (p.id, p)));
	// 	}

	// 	let mut threads_to_fetch = vec![];
	// 	let mut posts_to_fetch = vec![];
	// 	let mut fetching_posts = HashSet::new();
	// 	let mut fetch_post = |id: u64| {
	// 		if fetching_posts.insert(id) {
	// 			posts_to_fetch
	// 				.push(util::fetch_json(format!("/api/json/posts/{}", id)));
	// 		}
	// 	};
	// 	for thread in from_websocket.values() {
	// 		// Handle missing threads on index pages
	// 		if !loc.is_thread() && !threads.contains_key(&thread.thread) {
	// 			threads_to_fetch.push(util::fetch_json(format!(
	// 				"/api/json/threads/{}/-5",
	// 				thread.thread
	// 			)));
	// 		}

	// 		for id in thread.recent_posts.keys() {
	// 			if !posts.contains_key(id) {
	// 				fetch_post(*id);
	// 			}
	// 		}
	// 		for (id, open_post) in thread.open_posts.iter() {
	// 			if posts
	// 				.get(&id)
	// 				.map(|p| open_post.has_image && p.image.is_none())
	// 				.unwrap_or(true)
	// 			{
	// 				fetch_post(*id);
	// 			}
	// 		}
	// 	}

	// 	if threads_to_fetch.is_empty() && posts_to_fetch.is_empty() {
	// 		self.complete_feed_sync(loc, flags, threads, posts, from_websocket);
	// 		return;
	// 	}

	// 	self.feed_sync = FeedSync::Syncing {
	// 		loc: loc.clone(),
	// 		flags,
	// 		threads,
	// 		posts,
	// 		patches: from_websocket,
	// 	};

	// 	async fn run_fetches(
	// 		loc: Location,
	// 		threads: Vec<impl Future<Output = util::Result<ThreadWithPosts>>>,
	// 		posts: Vec<impl Future<Output = util::Result<Post>>>,
	// 	) {
	// 		use futures::future::{try_join, try_join_all};

	// 		Agent::dispatcher().send(Request::SyncFeed {
	// 			loc,
	// 			result: try_join(try_join_all(threads), try_join_all(posts))
	// 				.await,
	// 		});
	// 	}
	// 	wasm_bindgen_futures::spawn_local(run_fetches(
	// 		loc,
	// 		threads_to_fetch,
	// 		posts_to_fetch,
	// 	));
	// }

	// TODO: port
	// /// Complete syncing the feed using the received patch set
	// fn complete_feed_sync(
	// 	&mut self,
	// 	loc: Location,
	// 	mut flags: u8,
	// 	threads: HashMap<u64, Thread>,
	// 	posts: HashMap<u64, Post>,
	// 	patches: HashMap<u64, FeedData>,
	// ) {
	// 	// Apply patches received from the websocket to open posts received from
	// 	// JSON
	// 	write(|s| {
	// 		for patch in patches.values() {
	// 			for (id, created_on) in patch.recent_posts.iter() {
	// 				if let Some(p) = s.posts.get_mut(&id) {
	// 					p.created_on = *created_on;
	// 				}
	// 			}
	// 			for (id, op) in patch.open_posts.iter() {
	// 				if let Some(p) = s.posts.get_mut(&id) {
	// 					p.created_on = op.created_on;
	// 					p.body = (*op.body).clone();
	// 					if let Some(img) = &mut p.image {
	// 						img.spoilered = op.image_spoilered;
	// 					}
	// 				}
	// 			}
	// 		}
	// 	});

	// 	self.feed_sync = FeedSync::Synced {
	// 		feed: loc.feed.as_u64(),
	// 		patches,
	// 	};

	// 	// Trigger these updates in hierarchical order to make any upper level
	// 	// components switch rendering modes and not cause needless updates
	// 	// on deleted child components.
	// 	//
	// 	// Buffer and dedup hooks to be fired and handlers to be notified until
	// 	// update is complete.
	// 	let mut changes = vec![];
	// 	let mut changes_set = HashSet::new();
	// 	let mut add_hook = |h: Change| {
	// 		if changes_set.insert(h) {
	// 			changes.push(h);
	// 		}
	// 	};

	// 	flags |= FETCHED_JSON | NO_TRIGGER;
	// 	self.set_location(loc, flags);
	// 	add_hook(Change::Location);

	// 	write(|s| {
	// 		add_hook(Change::ThreadList);
	// 		for (id, _) in s.threads.drain() {
	// 			add_hook(Change::Thread(id));
	// 		}
	// 		for (id, _) in s.posts.drain() {
	// 			add_hook(Change::Post(id));
	// 		}

	// 		for id in threads.keys() {
	// 			add_hook(Change::Thread(*id));
	// 		}
	// 		s.threads = threads;

	// 		for p in posts.values() {
	// 			add_hook(Change::Post(p.id));
	// 			s.register_post_location(p);
	// 		}
	// 		s.posts = posts;
	// 	});

	// 	// Dedup hooked handlers to trigger
	// 	let mut sent = HashSet::with_capacity(changes.len());
	// 	for c in changes {
	// 		if let Some(reg) = self.hooks.get_by_key(&c) {
	// 			for r in reg.iter() {
	// 				if !sent.contains(r) {
	// 					sent.insert(*r);
	// 					self.link.respond(*r, ());
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	// TODO: port
	// /// Fetch feed data from JSON API
	// // TODO: fetch several pages at once for thread fetches where page!=0
	// fn fetch_feed_data(&mut self, new: Location, old: &Location, flags: u8) {
	// 	let new_feed = new.feed.as_u64();

	// 	// Clear any previous feed sync state, if feed changed
	// 	match &self.feed_sync {
	// 		// If feed did not change, this is a page navigation within the
	// 		// same feed. Keep the init data as there won't be any new received.
	// 		FeedSync::WebsocketReceived { feed, .. }
	// 		| FeedSync::Synced { feed, .. }
	// 			if *feed == new_feed =>
	// 		{
	// 			()
	// 		}
	// 		_ => {
	// 			self.feed_sync = Default::default();
	// 		}
	// 	};

	// 	// Start the websocket syncing process
	// 	if old.feed.as_u64() != new_feed {
	// 		use crate::connection::{Connection, Request};

	// 		Connection::dispatcher().send(Request::Synchronize(new_feed));
	// 	}

	// 	util::with_logging(|| {
	// 		use anyhow::Error;
	// 		use yew::{
	// 			format::{Json, Nothing},
	// 			services::fetch::{FetchService, Request, Response},
	// 		};

	// 		self.fetch_task = match new.feed.clone() {
	// 			FeedID::Index | FeedID::Catalog => FetchService::fetch(
	// 				Request::get("/api/json/index").body(Nothing).unwrap(),
	// 				self.link.callback(
	// 					move |res: Response<
	// 						Json<Result<Vec<ThreadWithPosts>, Error>>,
	// 					>| {
	// 						let (h, body) = res.into_parts();
	// 						match body {
	// 							Json(Ok(body)) => Message::FetchedThreadIndex {
	// 								data: body,
	// 								flags,
	// 								loc: new.clone(),
	// 							},
	// 							_ => Message::FetchFailed(format!(
	// 								concat!(
	// 									"error fetching index JSON: ",
	// 									"{} {:?}"
	// 								),
	// 								h.status, body,
	// 							)),
	// 						}
	// 					},
	// 				),
	// 			)?,
	// 			FeedID::Thread { id, page } => FetchService::fetch(
	// 				Request::get(&format!("/api/json/threads/{}/{}", id, page))
	// 					.body(Nothing)
	// 					.unwrap(),
	// 				self.link.callback(
	// 					move |res: Response<
	// 						Json<Result<ThreadWithPosts, Error>>,
	// 					>| {
	// 						let (h, body) = res.into_parts();
	// 						match body {
	// 							Json(Ok(body)) => {
	// 								// Convert -1 (last page) to actual page
	// 								// number
	// 								let mut loc = new.clone();
	// 								loc.feed = FeedID::Thread {
	// 									id: body.thread_data.id,
	// 									page: body.thread_data.page as i32,
	// 								};

	// 								Message::FetchedThread {
	// 									loc,
	// 									flags,
	// 									data: body,
	// 								}
	// 							}
	// 							_ => Message::FetchFailed(format!(
	// 								concat!(
	// 									"error fetching thread {} page {}",
	// 									" JSON: {} {:?}"
	// 								),
	// 								id, page, h.status, body,
	// 							)),
	// 						}
	// 					},
	// 				),
	// 			)?,
	// 			FeedID::Unset => unreachable!("move to Unset FeedID requested"),
	// 		}
	// 		.into();

	// 		Ok(())
	// 	})
	// }

	// TODO: port
	// fn process_successful_feed_fetch(
	// 	&mut self,
	// 	loc: Location,
	// 	threads: Vec<ThreadWithPosts>,
	// 	flags: u8,
	// ) {
	// 	debug_log!("fetched", threads);
	// 	self.fetch_task = None;
	// 	self.merge_feed_data(FeedSync::JSONFetched {
	// 		loc,
	// 		flags,
	// 		data: threads,
	// 	});
	// }
}

/// Navigate to the app to a different location
pub fn navigate_to(loc: Location) {
	Agent::dispatcher().send(Request::NavigateTo {
		loc,
		flags: PUSH_STATE | SET_STATE,
	});
}
