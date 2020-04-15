use crate::{post::image_search::Provider, util};
use protocol::{debug_log, post_body, AuthKey, DoubleSetMap, Image, SetMap};
use serde::{Deserialize, Serialize};
use std::{
	collections::{HashMap, HashSet},
	hash::Hash,
	str,
};
use yew::{
	agent::{AgentLink, Bridge, Context, HandlerId},
	services::render::{RenderService, RenderTask},
	Callback, Component, ComponentLink,
};

// Key used to store AuthKey in local storage
const AUTH_KEY: &str = "auth_key";

// Key used to store Options in local storage
const OPTIONS_KEY: &str = "options";

// Location setting flags
const PUSH_STATE: u8 = 1;
const SET_STATE: u8 = 1 << 1;
const FETCHED_JSON: u8 = 1 << 2;
const NO_TRIGGER: u8 = 1 << 3;

#[derive(Serialize, Deserialize, Clone, Copy, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ImageExpansionMode {
	None,
	FitWidth,
	FitHeight,
	FitScreen,
}

// Global user-set options
#[derive(Serialize, Deserialize)]
#[serde(default)]
pub struct Options {
	pub forced_anonymity: bool,
	pub relative_timestamps: bool,
	pub hide_thumbnails: bool,
	pub work_mode: bool,
	pub reveal_image_spoilers: bool,
	pub expand_gif_thumbnails: bool,
	pub enabled_image_search: Vec<Provider>,
	pub image_expansion_mode: ImageExpansionMode,
	pub audio_volume: u8,
}

impl Default for Options {
	fn default() -> Self {
		Self {
			forced_anonymity: false,
			relative_timestamps: true,
			hide_thumbnails: false,
			work_mode: false,
			reveal_image_spoilers: false,
			expand_gif_thumbnails: false,
			audio_volume: 100,
			image_expansion_mode: ImageExpansionMode::FitWidth,
			enabled_image_search: [
				Provider::Google,
				Provider::Yandex,
				Provider::IQDB,
				Provider::Trace,
				Provider::ExHentai,
			]
			.iter()
			.copied()
			.collect(),
		}
	}
}

// Exported public server configurations
//
// TODO: Get config updates though websocket
#[derive(Serialize, Deserialize, Default)]
pub struct Configs {
	pub captcha: bool,
	pub mature: bool,
	pub prune_threads: bool,
	pub thread_expiry: u32,
	pub max_size: u64,
	pub default_lang: String,
	pub default_theme: String,
	pub image_root_override: String,
	pub links: HashMap<String, String>,
}

// Stored separately from the agent to avoid needless serialization of post data
// on change propagation. The entire application has read-only access to this
// singleton. Writes have to be coordinated through the agent to ensure
// propagation.
#[derive(Default)]
pub struct State {
	// Location the app is currently navigated to
	pub location: Location,

	// Exported public server configurations
	pub configs: Configs,

	// All registered threads
	pub threads: HashMap<u64, Thread>,

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

	// Global user-set options
	pub options: Options,

	// Posts this user has made
	// TODO: Menu option to mark any post as mine
	// TODO: Persistance to indexedDB
	pub mine: HashSet<u64>,
}

protocol::gen_global! {pub, , State}

// Thread information container
#[derive(Serialize, Deserialize, Debug)]
pub struct Thread {
	pub id: u64,
	pub page: u32,
	pub last_page: u32,

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
	hooks: DoubleSetMap<Change, HandlerId>,
	fetch_task: Option<yew::services::fetch::FetchTask>,
	render_task: Option<RenderTask>,
}

#[derive(Serialize, Deserialize)]
pub enum Request {
	// Subscribe to updates of a value type
	NotifyChange(Change),

	// Set the client authorization key
	SetAuthKey(AuthKey),

	// Fetch feed data
	FetchFeed(Location),

	// Navigate to the app to a different feed
	NavigateTo { loc: Location, flags: u8 },
}

// Selective changes of global state to be notified on
#[derive(Serialize, Deserialize, Eq, PartialEq, Hash, Copy, Clone)]
pub enum Change {
	// Change of location the app is navigated to
	Location,

	// Auth key has been set by user
	AuthKey,

	// Change to any field of Options
	Options,

	// Change to any field of the Configs
	Configs,

	// Subscribe to changes of the list of threads
	ThreadList,

	// Subscribe to thread data changes, excluding the post content level.
	// This includes changes to the post set of threads.
	Thread(u64),

	// Subscribe to any changes to a post
	Post(u64),
}

// Abstraction over AgentLink and ComponentLink
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

// Helper for storing a hook into state updates in the client struct
pub struct HookBridge {
	#[allow(unused)]
	bridge: Box<dyn Bridge<Agent>>,
}

// Crate hooks into
pub fn hook<L, F>(link: &L, changes: &[Change], f: F) -> HookBridge
where
	L: Link,
	F: Fn(()) -> L::Message + 'static,
{
	use yew::agent::Bridged;

	let mut b = HookBridge {
		bridge: Agent::bridge(link.make_callback(f)),
	};
	for c in changes {
		b.bridge.send(Request::NotifyChange(*c))
	}
	b
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
	type Output = ();

	fn create(link: AgentLink<Self>) -> Self {
		util::add_static_listener(
			util::window(),
			"popstate",
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
				self.set_location(Location::from_path(), SET_STATE)
			}
		}
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		match req {
			Request::NotifyChange(h) => self.hooks.insert(h, id),
			Request::SetAuthKey(mut key) => util::with_logging(|| {
				write_auth_key(&mut key)?;
				write(|s| s.auth_key = key);
				self.trigger(Change::AuthKey);
				Ok(())
			}),
			Request::NavigateTo { loc, flags } => self.set_location(loc, flags),
			Request::FetchFeed(loc) => self.fetch_feed_data(loc, 0),
		};
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.hooks.remove_by_value(&id);
	}
}
impl Agent {
	// Send change notification to hooked clients
	fn trigger(&self, h: Change) {
		if let Some(subs) = self.hooks.get_by_key(&h) {
			for id in subs.iter() {
				self.link.respond(*id, ());
			}
		}
	}

	// Set app location and propagate changes
	fn set_location(&mut self, new: Location, flags: u8) {
		write(|s| {
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
					(FeedID::Thread { .. }, _) | (_, FeedID::Thread { .. }) => {
						true
					}

					// Catalog and Index transition do not need a fetch
					_ => false,
				};
			if need_fetch {
				self.fetch_feed_data(new, flags);
				return;
			}

			if flags & SET_STATE != 0 {
				s.location = new.clone();
				if flags & NO_TRIGGER != 0 {
					self.trigger(Change::Location);
				}
				if let Some(f) = new.focus.clone() {
					self.render_task = RenderService::new()
						.request_animation_frame(
							self.link
								.callback(move |_| Message::Focus(f.clone())),
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

		// Trigger these updates in hierarchical order to make any upper level
		// components to switch rendering modes and not cause needless updates
		// on deleted child components.
		//
		// Buffer and dedup hooks to be fired and handlers to be notified until
		// update is complete.
		let mut changes = vec![];
		let mut changes_set = HashSet::new();
		let mut add_hook = |h: Change| {
			if !changes_set.contains(&h) {
				changes.push(h);
				changes_set.insert(h);
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
			for (id, _) in s.posts.drain() {
				add_hook(Change::Thread(id));
			}
			s.posts_by_thread_page.clear();

			for t in threads {
				let t_id = t.thread_data.id;
				add_hook(Change::Thread(t_id));
				s.threads.insert(t_id, t.thread_data);
				for p in t.posts {
					add_hook(Change::Post(p.id));
					s.posts_by_thread_page.insert((t_id, p.page), p.id);
					s.posts.insert(p.id, p);
				}
			}
		});

		// Dedup hooked handlers to trigger
		let mut handlers = Vec::with_capacity(changes.len());
		let mut handlers_set = HashSet::with_capacity(changes.len());
		for c in changes {
			if let Some(reg) = self.hooks.get_by_key(&c) {
				for r in reg.iter() {
					if !handlers_set.contains(r) {
						handlers_set.insert(*r);
						handlers.push(*r);
					}
				}
			}
		}
		for h in handlers {
			self.link.respond(h, ());
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

	write(|s| {
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

		// Read saved options, if any
		if let Some(v) = ls.get_item(OPTIONS_KEY).unwrap() {
			if let Ok(opts) = serde_json::from_str(&v) {
				s.options = opts;
			}
		}

		// Manage scrolling ourselves
		util::window()
			.history()?
			.set_scroll_restoration(web_sys::ScrollRestoration::Manual)?;

		// Read configs from JSON embedded in the HTML
		s.configs = serde_json::from_str(
			&util::document()
				.get_element_by_id("config-data")
				.ok_or("inline configs not found")?
				.inner_html(),
		)?;

		Ok(())
	})
}
