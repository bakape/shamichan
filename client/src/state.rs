use crate::util;
use protocol::*;
use serde::{Deserialize, Serialize};
use std::{
	collections::{HashMap, HashSet},
	hash::Hash,
	str,
};
use yew::{
	agent::{AgentLink, Context, HandlerId},
	services::render::{RenderService, RenderTask},
};

// Key used to store AuthKey in local storage
const AUTH_KEY: &str = "auth_key";

// Location setting flags
const PUSH_STATE: u8 = 1;
const SET_STATE: u8 = 1 << 1;
const FETCHED_JSON: u8 = 1 << 2;

// Stored separately from the agent to avoid needless serialization of post data
// on change propagation. The entire application has read-only access to this
// singleton. Writes have to be coordinated through the agent to ensure
// propagation.
#[derive(Default)]
pub struct State {
	// Location the app is currently navigated to
	pub location: Location,

	// All registered threads
	pub threads: HashMap<u64, Thread>,

	// Page count of threads
	pub page_counts: HashMap<u64, u32>,

	// All registered posts from any sources
	//
	// TODO: Some kind of post eviction algorithm.
	// For now posts are never removed from memory for easing implementation of
	// a more persistent cross-feed UI.
	pub posts: HashMap<u64, Post>,

	// Map for quick lookup of post IDs by a (thread, page) tuple
	pub posts_by_thread_page: SetMap<(u64, u32), u64>,

	// Authentication key
	pub auth_key: AuthKey,

	// Posts this user has made
	// TODO: Menu option to mark any post as mine
	// TODO: Persistance to indexedDB
	pub mine: HashSet<u64>,
}

impl State {
	fn insert_post(&mut self, p: Post) {
		self.posts_by_thread_page.insert((p.thread, p.page), p.id);
		match self.page_counts.get_mut(&p.thread) {
			Some(l) => {
				if &p.page >= l {
					*l = p.page + 1;
				}
			}
			None => {
				self.page_counts.insert(p.thread, p.page + 1);
			}
		};
		self.posts.insert(p.id, p);
	}

	// Clear all thread data.
	//
	// Post data is still retained for now to ease UI building until more
	// concrete lifetime requirements are determined.
	fn clear_threads(&mut self) {
		self.threads.clear();
		self.page_counts.clear();
	}
}

super::gen_global! {pub, State, get, get_mut}

// Thread information container
#[derive(Serialize, Deserialize, Debug)]
pub struct Thread {
	pub id: u64,
	pub page: u32,

	pub subject: String,
	pub tags: Vec<String>,

	pub bumped_on: u32,
	pub created_on: u32,
	pub post_count: u64,
	pub image_count: u64,
}

// Post data
#[derive(Serialize, Deserialize, Debug)]
pub struct Post {
	pub id: u64,
	pub page: u32,
	pub thread: u64,

	pub created_on: u32,
	pub open: bool,

	pub name: Option<String>,
	pub trip: Option<String>,
	pub flag: Option<String>,
	pub sage: bool,

	pub body: Option<post_body::Node>,
	pub image: Option<Image>,
}

// Decodes thread data received from the server as JSON
#[derive(Serialize, Deserialize, Debug)]
pub struct ThreadDecoder {
	#[serde(flatten)]
	thread_data: Thread,

	posts: Vec<Post>,
}

// Global state storage and propagation agent
pub struct Agent {
	link: AgentLink<Self>,

	// Subscriber registry
	subscribers: DoubleSetMap<Subscription, HandlerId>,

	fetch_task: Option<yew::services::fetch::FetchTask>,

	render_task: Option<RenderTask>,
}

#[derive(Serialize, Deserialize)]
pub enum Request {
	// Subscribe to updates of a value type
	Subscribe(Subscription),

	// Set the client authorization key
	SetAuthKey(AuthKey),

	// Fetch feed data
	FetchFeed(Location),

	// Navigate to the app to a different feed
	//
	// TODO: also focus post after render
	NavigateTo { loc: Location, flags: u8 },
}

// Value changes to subscribe to
#[derive(Serialize, Deserialize, Eq, PartialEq, Hash, Clone, Debug)]
pub enum Subscription {
	// Change of location the app is navigated to
	LocationChange,

	// Auth key has been set by user
	AuthKeyChange,

	// Subscribe to any changes to a post
	PostChange(u64),

	// Subscribe to thread data changes, excluding the post content level.
	// This includes changes to the post set of threads.
	ThreadChange(u64),

	// Subscribe to changes of the list of threads
	ThreadListChange,

	// Change to any field of Configs
	ConfigsChange,
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum Response {
	// Change of location the app is navigated to
	LocationChange { old: Location, new: Location },

	// Response with no payload that simply identifies what Subscription
	// triggered it
	NoPayload(Subscription),
}

// Identifies a global index or thread feed
#[derive(Serialize, Deserialize, Debug, PartialEq, Eq, Clone)]
pub enum FeedID {
	Index,
	Catalog,
	Thread { id: u64, page: i32 },
}

impl Default for FeedID {
	fn default() -> FeedID {
		FeedID::Index
	}
}

// Post or page margin to scroll to
#[derive(Serialize, Deserialize, Clone, Eq, PartialEq, Debug)]
pub enum Focus {
	Top,
	Bottom,
	Post(u64),
}

impl Default for Focus {
	fn default() -> Focus {
		Focus::Top
	}
}

// Location the app can navigate to
#[derive(Default, Serialize, Deserialize, Clone, Eq, PartialEq, Debug)]
pub struct Location {
	pub feed: FeedID,

	// Focus a post after a successful render
	pub focus: Option<Focus>,
}

impl Location {
	fn from_path() -> Location {
		let loc = util::window().location();
		let path = loc.pathname().unwrap_or_default();
		let split: Vec<&str> = path.split('/').collect();
		Location {
			feed: match (split.get(1), split.len()) {
				(Some(&"threads"), 4) => {
					macro_rules! parse {
						($i:expr) => {
							split.get($i).map(|s| s.parse().ok()).flatten()
						};
					}

					match (parse!(2), parse!(3)) {
						(Some(thread), Some(page)) => FeedID::Thread {
							id: thread,
							page: page,
						},
						_ => FeedID::Index,
					}
				}
				(Some(&"catalog"), _) => FeedID::Catalog,
				_ => FeedID::Index,
			},
			focus: loc
				.hash()
				.ok()
				.map(|h| match h.as_str() {
					"#top" => Some(Focus::Top),
					"#bottom" => Some(Focus::Bottom),
					_ => match h.get(..3) {
						Some("#p-") => h
							.get(3..)
							.map(|s| s.parse().ok())
							.flatten()
							.map(|id| Focus::Post(id)),
						_ => None,
					},
				})
				.flatten(),
		}
	}

	fn path(&self) -> String {
		let mut w: String = match &self.feed {
			FeedID::Index => "/".into(),
			FeedID::Catalog => "/catalog".into(),
			FeedID::Thread { id, page } => format!("/threads/{}/{}", id, page),
		};
		if let Some(f) = &self.focus {
			match f {
				Focus::Bottom => {
					w += "#bottom";
				}
				Focus::Top => {
					w += "#top";
				}
				Focus::Post(id) => {
					use std::fmt::Write;

					write!(w, "#p-{}", id).unwrap();
				}
			}
		}
		w
	}

	// Returns, if this is a single thread page
	pub fn is_thread(&self) -> bool {
		matches!(self.feed, FeedID::Thread { .. })
	}
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
	type Reach = Context;
	type Message = Message;
	type Input = Request;
	type Output = Response;

	fn create(link: AgentLink<Self>) -> Self {
		debug_log!("adding popstate listener");
		util::add_static_listener(
			util::window(),
			"popstate",
			link.callback(|_: web_sys::Event| Message::PoppedState),
		);

		Self {
			link,
			subscribers: DoubleSetMap::default(),
			fetch_task: None,
			render_task: None,
		}
	}

	fn update(&mut self, msg: Self::Message) {
		match msg {
			Message::FetchedThreadIndex { loc, data, flags } => {
				self.process_successful_feed_fetch(loc, data, flags);
			}
			Message::FetchedThread { loc, data, flags } => {
				// Option<T> implements IntoIterator<Item=T>
				self.process_successful_feed_fetch(loc, Some(data), flags);
			}
			Message::FetchFailed(s) => {
				util::log_error(&s);
				util::alert(&s);
				self.fetch_task = None;
			}
			Message::Focus(f) => {
				use util::document;
				use wasm_bindgen::JsCast;
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
						Focus::Top => banner_height(),
						Focus::Bottom => document()
							.document_element()
							.map(|el| el.scroll_height())
							.unwrap_or_default() as f64,
						Focus::Post(id) => document()
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
			Message::PoppedState => {
				debug_log!("popped state", util::window().location().href());
				self.set_location(Location::from_path(), SET_STATE)
			}
		}
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		match req {
			Request::Subscribe(t) => self.subscribers.insert(t, id),
			Request::SetAuthKey(mut key) => util::with_logging(|| {
				write_auth_key(&mut key)?;
				get_mut().auth_key = key;
				self.send_change_no_payload(Subscription::AuthKeyChange);
				Ok(())
			}),
			Request::NavigateTo { loc, flags } => self.set_location(loc, flags),
			Request::FetchFeed(loc) => self.fetch_feed_data(loc, 0),
		};
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove_by_value(&id);
	}
}

impl Agent {
	// Send change notification to all subscribers of sub
	fn send_change(&self, sub: Subscription, res: Response) {
		if let Some(subs) = self.subscribers.get_by_key(&sub) {
			for id in subs.iter() {
				self.link.respond(*id, res.clone());
			}
		}
	}

	// Set app location and propagate changes
	fn set_location(&mut self, new: Location, flags: u8) {
		let s = get_mut();
		debug_log!(
			"set_location",
			format!("{:?} -> {:?}, flags={}", s.location, new, flags)
		);

		let old = s.location.clone();
		let need_fetch = flags & FETCHED_JSON == 0
			&& match (&old.feed, &new.feed) {
				(
					FeedID::Thread {
						id: old_id,
						page: old_page,
					},
					FeedID::Thread {
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
				(FeedID::Thread { .. }, _) | (_, FeedID::Thread { .. }) => true,

				// Catalog and Index transition do not need a fetch
				_ => false,
			};
		debug_log!("need_fetch", need_fetch);
		if need_fetch {
			self.fetch_feed_data(new, flags);
			return;
		}

		if flags & SET_STATE != 0 {
			debug_log!("setting location", new);

			s.location = new.clone();
			self.send_change(
				Subscription::LocationChange,
				Response::LocationChange {
					old,
					new: new.clone(),
				},
			);
			if let Some(f) = new.focus.clone() {
				self.render_task = RenderService::new()
					.request_animation_frame(
						self.link.callback(move |_| Message::Focus(f.clone())),
					)
					.into();
			}
		}

		if flags & PUSH_STATE != 0 {
			debug_log!("pushing history state", new);

			// TODO: Set last scroll position on back and hash navigation using
			// replace_state()
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

	// Send change notification to all subscribers of sub with no payload
	fn send_change_no_payload(&self, sub: Subscription) {
		let res = Response::NoPayload(sub.clone());
		self.send_change(sub, res);
	}

	// Fetch feed data from JSON API!
	fn fetch_feed_data(&mut self, loc: Location, flags: u8) {
		util::with_logging(|| {
			use anyhow::Error;
			use yew::{
				format::{Json, Nothing},
				services::fetch::{FetchService, Request, Response},
			};

			self.fetch_task = match loc.feed.clone() {
				FeedID::Index | FeedID::Catalog => FetchService::new().fetch(
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
				FeedID::Thread { id, page } => FetchService::new().fetch(
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
		debug_log!("fetched", threads);
		self.fetch_task = None;

		// Trigger these updates first to cause any upper level components
		// to switch rendering modes and not cause needless updates on deleted
		// child components.
		flags |= FETCHED_JSON;
		self.set_location(loc, flags);
		self.send_change_no_payload(Subscription::ThreadListChange);

		let s = get_mut();
		s.clear_threads();
		for t in threads {
			let id = t.thread_data.id;
			self.send_change_no_payload(Subscription::ThreadChange(id));
			s.threads.insert(id, t.thread_data);
			for p in t.posts {
				self.send_change_no_payload(Subscription::PostChange(p.id));
				s.insert_post(p);
			}
		}
	}
}

// Navigate to the app to a different location
pub fn navigate_to(loc: Location) {
	use yew::agent::Dispatched;

	Agent::dispatcher().send(Request::NavigateTo {
		loc,
		flags: PUSH_STATE | SET_STATE,
	});
}

fn write_auth_key(key: &mut AuthKey) -> util::Result {
	let mut buf: [u8; 88] =
		unsafe { std::mem::MaybeUninit::uninit().assume_init() };
	base64::encode_config_slice(key, base64::STANDARD, &mut buf);

	util::local_storage()
		.set_item(AUTH_KEY, unsafe { str::from_utf8_unchecked(&buf) })?;
	Ok(())
}

// Initialize application state
pub fn init() -> util::Result {
	fn create_auth_key() -> util::Result<AuthKey> {
		let mut key = AuthKey::default();
		util::window()
			.crypto()?
			.get_random_values_with_u8_array(key.as_mut())?;
		write_auth_key(&mut key)?;
		Ok(key)
	}

	let mut s = get_mut();
	s.location = Location::from_path();

	// Read saved or generate a new authentication key
	let ls = util::local_storage();
	s.auth_key = match ls.get_item(AUTH_KEY).unwrap() {
		Some(v) => {
			let mut key = AuthKey::default();
			match base64::decode_config_slice(
				&v,
				base64::STANDARD,
				key.as_mut(),
			) {
				Ok(_) => key,
				_ => create_auth_key()?,
			}
		}
		None => create_auth_key()?,
	};

	// Manage scrolling ourselves
	util::window()
		.history()?
		.set_scroll_restoration(web_sys::ScrollRestoration::Manual)?;

	Ok(())
}
