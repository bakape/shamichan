pub mod agent;
pub mod key_pair;
pub mod location;
pub mod options;

pub use agent::{
	hook, navigate_to, Agent, Change, HookBridge, Link, Message, Request,
};
pub use location::{FeedID, Focus, Location};
pub use options::{ImageExpansionMode, Options};

use crate::util;
use key_pair::KeyPair;
use protocol::{
	payloads::{post_body::Node, Image},
	util::SetMap,
};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};

/// Exported public server configurations
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

/// Optional flags and contents for creating new posts (including OPs)
#[derive(Default)]
pub struct NewPostOpts {
	pub sage: bool,
	pub name: String,
	// TODO: staff titles
}

/// Stored separately from the agent to avoid needless serialization of post data
/// on change propagation. The entire application has read-only access to this
/// singleton. Writes have to be coordinated through the agent to ensure
/// propagation.
#[derive(Default)]
pub struct State {
	/// Location the app is currently navigated to
	pub location: Location,

	/// Exported public server configurations
	pub configs: Configs,

	/// All registered threads
	pub threads: HashMap<u64, Thread>,

	/// All registered posts from any sources
	//
	// TODO: Some kind of post eviction algorithm.
	// For now posts are never removed from memory for easing implementation of
	// a more persistent cross-feed UI.
	pub posts: HashMap<u64, Post>,

	/// Map for quick lookup of post IDs by a (thread, page) tuple
	pub posts_by_thread_page: SetMap<(u64, u32), u64>,

	/// Authentication key pair
	pub key_pair: KeyPair,

	/// Public key UUID stored on the server
	pub key_id: Option<uuid::Uuid>,

	/// Global user-set options
	pub options: Options,

	/// ID of currently open allocated post, if any
	pub open_post_id: Option<u64>,

	/// Posts this user has made
	// TODO: Menu option to mark any post as mine
	// TODO: Persistance to indexedDB
	pub mine: HashSet<u64>,

	/// Optional flags and contents for creating new posts (including OPs)
	pub new_post_opts: NewPostOpts,
}

impl State {
	/// Insert a post into the registry
	fn register_post(&mut self, p: Post) {
		self.posts_by_thread_page.insert((p.thread, p.page), p.id);
		self.posts.insert(p.id, p);
	}
}

protocol::gen_global! {
	State {
		pub fn read();
		pub fn write();
	}
}

/// Thread information container
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

/// Post data
#[derive(Serialize, Deserialize, Debug, Default)]
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

	pub body: Node,
	pub image: Option<Image>,
}

/// Initialize application state
pub async fn init() -> util::Result {
	let kp = KeyPair::load().await?;

	write(|s| {
		s.key_pair = kp;
		s.location = Location::from_path();
		s.options.load();

		// Manage scrolling ourselves because of the dynamic nature of page
		// generation
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
