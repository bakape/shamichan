use crate::{post::image_search::Provider, util};
use protocol::*;
use serde::{Deserialize, Serialize};
use std::{
	collections::{HashMap, HashSet},
	hash::Hash,
	str,
};
use yew::agent::{AgentLink, Context, HandlerId};

// Key used to store AuthKey in local storage
const AUTH_KEY: &str = "auth_key";

// Key used to store Options in local storage
const OPTIONS_KEY: &str = "options";

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

// Stored separately from the agent to avoid needless serialization on change
// propagation. The entire application has read-only access to this singleton.
// Writes have to be coordinated through the agent to ensure propagation.
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

	// Global user-set options
	pub options: Options,

	// Exported public server configurations
	pub configs: Configs,

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
	NavigateTo(Location),
}

// Value changes to subscribe to
#[derive(Serialize, Deserialize, Eq, PartialEq, Hash, Clone)]
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

	// Change to any field of Options
	OptionsChange,

	// Change to any field of Configs
	ConfigsChange,
}

#[derive(Serialize, Deserialize, Clone)]
pub enum Response {
	// Change of location the app is navigated to
	LocationChange { old: Location, new: Location },

	// Response with no payload that simply identifies what Subscription
	// triggered it
	NoPayload(Subscription),
}

// Identifies a thread feed
#[derive(Serialize, Deserialize, Debug, PartialEq, Eq, Clone)]
pub struct ThreadFeedID {
	pub id: u64,
	pub page: i32,
}

// Identifies a global index or thread feed
#[derive(Serialize, Deserialize, Debug, PartialEq, Eq, Clone)]
pub enum FeedID {
	Index,
	Thread(ThreadFeedID),
}

impl Default for FeedID {
	fn default() -> FeedID {
		FeedID::Index
	}
}

// Location the app can navigate to
#[derive(Default, Serialize, Deserialize, Clone, Eq, PartialEq)]
pub struct Location {
	pub feed: FeedID,

	// Focus a post after a successful render
	pub focus_post: Option<u64>,
}

impl Location {
	fn from_path() -> Location {
		let loc = util::window().location();
		let path = loc.pathname().unwrap_or_default();
		let split: Vec<&str> = path.split('/').collect();
		Location {
			feed: match (split.get(0), split.len()) {
				(Some(&"threads"), 3) => {
					macro_rules! parse {
						($i:expr) => {
							split.get($i).map(|s| s.parse().ok()).flatten()
						};
					}

					match (parse!(1), parse!(2)) {
						(Some(thread), Some(page)) => {
							FeedID::Thread(ThreadFeedID {
								id: thread,
								page: page,
							})
						}
						_ => FeedID::Index,
					}
				}
				_ => FeedID::Index,
			},
			focus_post: loc
				.hash()
				.ok()
				.map(|h| h.get(3..).map(|s| s.parse().ok()))
				.flatten()
				.flatten(),
		}
	}
}

pub enum Message {
	FetchedThreadIndex {
		loc: Location,
		data: Vec<ThreadDecoder>,
	},
	FetchedThread {
		loc: Location,
		data: ThreadDecoder,
	},
	FetchFailed(String),
}

impl yew::agent::Agent for Agent {
	type Reach = Context;
	type Message = Message;
	type Input = Request;
	type Output = Response;

	fn create(link: AgentLink<Self>) -> Self {
		Self {
			link,
			subscribers: DoubleSetMap::default(),
			fetch_task: None,
		}
	}

	fn update(&mut self, msg: Self::Message) {
		match msg {
			Message::FetchedThreadIndex { loc, data } => {
				self.process_successful_feed_fetch(loc, data);
			}
			Message::FetchedThread { loc, data } => {
				// Option<T> implements IntoIterator<Item=T>
				self.process_successful_feed_fetch(loc, Some(data));
			}
			Message::FetchFailed(s) => {
				util::log_error(&s);
				util::alert(&s);
				self.fetch_task = None;
			}
		}
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		match req {
			Request::Subscribe(t) => {
				self.subscribers.insert(t, id);
			}
			Request::SetAuthKey(key) => {
				get_mut().auth_key = key;
				self.send_change_no_payload(Subscription::AuthKeyChange);
			}
			Request::NavigateTo(loc) => self.set_location(loc),
			Request::FetchFeed(loc) => self.fetch_feed_data(loc),
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
	fn set_location(&mut self, new: Location) {
		let s = get_mut();
		if s.location != new {
			let old = s.location.clone();
			s.location = new.clone();

			// Page number corrections do not need a refetch
			if old.feed != new.feed
				&& match (&old.feed, &new.feed) {
					(FeedID::Thread(old), FeedID::Thread(_)) => old.page != -1,
					_ => true,
				} {
				self.fetch_feed_data(new.clone());
			}

			self.send_change(
				Subscription::LocationChange,
				Response::LocationChange { old, new },
			);
			// TODO: History navigation
		}
	}

	// Send change notification to all subscribers of sub with no payload
	fn send_change_no_payload(&self, sub: Subscription) {
		let res = Response::NoPayload(sub.clone());
		self.send_change(sub, res);
	}

	// Fetch feed data from JSON API
	fn fetch_feed_data(&mut self, loc: Location) {
		util::with_logging(|| {
			use anyhow::Error;
			use yew::{
				format::{Json, Nothing},
				services::fetch::{FetchService, Request, Response},
			};

			self.fetch_task = match loc.feed.clone() {
				FeedID::Index => FetchService::new().fetch(
					Request::get("/api/json/index").body(Nothing).unwrap(),
					self.link.callback(
						move |res: Response<
							Json<Result<Vec<ThreadDecoder>, Error>>,
						>| {
							let (h, body) = res.into_parts();
							match body {
								Json(Ok(body)) => Message::FetchedThreadIndex {
									data: body,
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
				FeedID::Thread(feed) => FetchService::new().fetch(
					Request::get(&format!(
						"/api/json/threads/{}/{}",
						feed.id, feed.page
					))
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
									loc.feed = FeedID::Thread(ThreadFeedID {
										id: body.thread_data.id,
										page: body.thread_data.page as i32,
									});

									Message::FetchedThread { loc, data: body }
								}
								_ => Message::FetchFailed(format!(
									concat!(
										"error fetching thread {:?}",
										" JSON: {} {:?}"
									),
									feed, h.status, body,
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

	fn process_successful_feed_fetch<T>(&mut self, loc: Location, threads: T)
	where
		T: IntoIterator<Item = ThreadDecoder> + std::fmt::Debug,
	{
		debug_log!("fetched", threads);
		self.fetch_task = None;

		// Trigger these updates first to cause any upper level components
		// to switch rendering modes and not cause needless updates on deleted
		// child components.
		self.set_location(loc.clone());
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

		if let Some(_) = loc.focus_post {
			todo!("scroll to focused post after render, if any")
		}
	}
}

// Navigate to the app to a different location
pub fn navigate_to(loc: Location) {
	use yew::agent::Dispatched;

	Agent::dispatcher().send(Request::NavigateTo(loc));
}

fn write_auth_key(key: &mut AuthKey) {
	let mut buf: [u8; 88] =
		unsafe { std::mem::MaybeUninit::uninit().assume_init() };
	base64::encode_config_slice(key, base64::STANDARD, &mut buf);

	util::with_logging(|| {
		util::local_storage()
			.set_item(AUTH_KEY, unsafe { str::from_utf8_unchecked(&buf) })
			.map_err(|e| e.into())
	});
}

// Initialize application state
pub fn init() -> util::Result {
	fn create_auth_key() -> AuthKey {
		let mut key = AuthKey::default();
		util::window()
			.crypto()
			.unwrap()
			.get_random_values_with_u8_array(key.as_mut())
			.unwrap();
		write_auth_key(&mut key);
		key
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
				_ => create_auth_key(),
			}
		}
		None => create_auth_key(),
	};

	// Read saved options, if any
	if let Some(v) = ls.get_item(OPTIONS_KEY).unwrap() {
		if let Ok(opts) = serde_json::from_str(&v) {
			s.options = opts;
		}
	}

	// Read configs from JSON embedded in the HTML
	s.configs = serde_json::from_str(
		&util::document()
			.get_element_by_id("config-data")
			.ok_or("inline configs not found")?
			.inner_html(),
	)?;

	Ok(())
}
