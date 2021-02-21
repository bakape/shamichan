use super::{KeyPair, Location, Options};
use crate::util;
use common::{
	payloads::{Post, Thread},
	util::DoubleSetMap,
};
use std::{
	cell::{Ref, RefCell, RefMut},
	collections::{HashMap, HashSet},
	hint::unreachable_unchecked,
	rc::Rc,
};

/// Global application state
static mut STATE: *mut RefCell<State> = std::ptr::null_mut();

/// Get immutable reference to application state
#[inline]
pub(super) fn get_ref() -> Ref<'static, State> {
	match unsafe { STATE.as_ref() } {
		Some(s) => s,
		None => unsafe { unreachable_unchecked() },
	}
	.borrow()
}

/// Get mutable reference to application state
#[inline]
pub(super) fn get_mut() -> RefMut<'static, State> {
	match unsafe { STATE.as_mut() } {
		Some(s) => s,
		None => unsafe { unreachable_unchecked() },
	}
	.borrow_mut()
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
	//
	// TODO: Get config updates though websocket
	pub configs: common::config::Public,

	/// All registered threads
	pub threads: HashMap<u64, Thread>,

	/// All registered posts for the current feed
	pub posts: HashMap<u64, Post>,

	/// Map for quick lookup of post IDs by a (thread, page) tuple and vice
	/// versa
	pub posts_by_thread_page: DoubleSetMap<(u64, u32), u64>,

	/// Pages loaded for the current thread
	pub loaded_pages: HashSet<u32>,

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

	/// Tags already used on threads
	pub used_tags: Rc<Vec<String>>,

	/// Time correction between the server and client.
	/// Add to client-generated unix timestamps to correct them.
	pub time_correction: i32,
}

impl State {
	/// Insert a post into the registry
	pub(super) fn register_post(&mut self, p: Post) {
		self.posts_by_thread_page.insert((p.thread, p.page), p.id);
		self.posts.insert(p.id, p);
	}

	/// Get metainformation of a thread that must be synced.
	/// Panics on no thread found.
	pub(super) fn get_synced_thread(&self, id: &u64) -> &Thread {
		self.threads
			.get(id)
			.ok_or("no meta for synced thread")
			.unwrap()
	}
}

/// Initialize application state
#[cold]
pub async fn init() -> util::Result {
	let mut s = State::default();

	s.key_pair = KeyPair::load().await?;
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

	unsafe { STATE = Box::into_raw(Box::new(RefCell::new(s))) };

	Ok(())
}
