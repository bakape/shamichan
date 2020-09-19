use super::{write, FeedID, Focus, Location, Post, Thread};
use crate::util;
use protocol::{debug_log, payloads::ThreadCreationNotice, util::DoubleSetMap};
use serde::{Deserialize, Serialize};
use wasm_bindgen::JsCast;
use yew::{
	agent::{AgentLink, Bridge, Context, HandlerId},
	services::render::{RenderService, RenderTask},
	Callback, Component, ComponentLink,
};

/// Location setting flags
const PUSH_STATE: u8 = 1;
const SET_STATE: u8 = 1 << 1;
const FETCHED_JSON: u8 = 1 << 2;
const NO_TRIGGER: u8 = 1 << 3;

/// Decodes thread data received from the server as JSON
#[derive(Serialize, Deserialize, Debug)]
pub struct ThreadDecoder {
	#[serde(flatten)]
	thread_data: Thread,

	posts: Vec<Post>,
}

/// Global state storage and propagation agent
pub struct Agent {
	link: AgentLink<Self>,
	hooks: DoubleSetMap<Change, HandlerId>,
	fetch_task: Option<yew::services::fetch::FetchTask>,
	render_task: Option<RenderTask>,
}

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
	FetchedThreadIndex {
		loc: Location,
		data: Vec<ThreadDecoder>,
		flags: u8,
	},
	FetchedThread {
		loc: Location,
		data: ThreadDecoder,
		flags: u8,
	},
	FetchFailed(String),
	Focus(Focus),
	PoppedState,
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
		}
	}

	fn update(&mut self, msg: Self::Message) {
		use Message::*;

		match msg {
			FetchedThreadIndex { loc, data, flags } => {
				self.process_successful_feed_fetch(loc, data, flags);
			}
			FetchedThread { loc, data, flags } => {
				self.process_successful_feed_fetch(
					loc,
					std::iter::once(data),
					flags,
				);
			}
			FetchFailed(s) => {
				util::log_error(&s);
				util::alert(&s);
				self.fetch_task = None;
			}
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
			FetchFeed(loc) => self.fetch_feed_data(loc, 0),
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
							page: 0,
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
						created_on: n.time,
						open: true,
						..Default::default()
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
						use yew::agent::Dispatched;

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
				self.fetch_feed_data(new, flags);
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

	/// Fetch feed data from JSON API
	fn fetch_feed_data(&mut self, loc: Location, flags: u8) {
		util::with_logging(|| {
			use anyhow::Error;
			use yew::{
				format::{Json, Nothing},
				services::fetch::{FetchService, Request, Response},
			};

			self.fetch_task = match loc.feed.clone() {
				FeedID::Index | FeedID::Catalog => FetchService::fetch(
					Request::get("/api/json/index").body(Nothing).unwrap(),
					self.link.callback(
						move |res: Response<
							Json<Result<Vec<ThreadDecoder>, Error>>,
						>| {
							let (h, body) = res.into_parts();
							match body {
								Json(Ok(body)) => Message::FetchedThreadIndex {
									data: body,
									flags,
									loc: loc.clone(),
								},
								_ => Message::FetchFailed(format!(
									concat!(
										"error fetching index JSON: ",
										"{} {:?}"
									),
									h.status, body,
								)),
							}
						},
					),
				)?,
				FeedID::Thread { id, page } => FetchService::fetch(
					Request::get(&format!("/api/json/threads/{}/{}", id, page))
						.body(Nothing)
						.unwrap(),
					self.link.callback(
						move |res: Response<
							Json<Result<ThreadDecoder, Error>>,
						>| {
							let (h, body) = res.into_parts();
							match body {
								Json(Ok(body)) => {
									// Convert -1 (last page) to actual page
									// number
									let mut loc = loc.clone();
									loc.feed = FeedID::Thread {
										id: body.thread_data.id,
										page: body.thread_data.page as i32,
									};

									Message::FetchedThread {
										loc,
										flags,
										data: body,
									}
								}
								_ => Message::FetchFailed(format!(
									concat!(
										"error fetching thread {} page {}",
										" JSON: {} {:?}"
									),
									id, page, h.status, body,
								)),
							}
						},
					),
				)?,
			}
			.into();

			Ok(())
		})
	}

	fn process_successful_feed_fetch<T>(
		&mut self,
		loc: Location,
		threads: T,
		mut flags: u8,
	) where
		T: IntoIterator<Item = ThreadDecoder> + std::fmt::Debug,
	{
		use std::collections::HashSet;

		debug_log!("fetched", threads);
		self.fetch_task = None;

		// Trigger these updates in hierarchical order to make any upper level
		// components to switch rendering modes and not cause needless updates
		// on deleted child components.
		//
		// Buffer and dedup hooks to be fired and handlers to be notified until
		// update is complete.
		let mut changes = vec![];
		let mut changes_set = HashSet::new();
		let mut add_hook = |h: Change| {
			if changes_set.insert(h) {
				changes.push(h);
			}
		};

		flags |= FETCHED_JSON | NO_TRIGGER;
		self.set_location(loc, flags);
		add_hook(Change::Location);

		write(|s| {
			add_hook(Change::ThreadList);
			for (id, _) in s.threads.drain() {
				add_hook(Change::Thread(id));
			}

			for t in threads {
				let t_id = t.thread_data.id;
				add_hook(Change::Thread(t_id));
				s.threads.insert(t_id, t.thread_data);
				for p in t.posts {
					add_hook(Change::Post(p.id));
					s.register_post(p);
				}
			}
		});

		// Dedup hooked handlers to trigger
		let mut sent = HashSet::with_capacity(changes.len());
		for c in changes {
			if let Some(reg) = self.hooks.get_by_key(&c) {
				for r in reg.iter() {
					if !sent.contains(r) {
						sent.insert(*r);
						self.link.respond(*r, ());
					}
				}
			}
		}
	}
}

/// Navigate to the app to a different location
pub fn navigate_to(loc: Location) {
	use yew::agent::Dispatched;

	Agent::dispatcher().send(Request::NavigateTo {
		loc,
		flags: PUSH_STATE | SET_STATE,
	});
}
