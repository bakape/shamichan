#![allow(dead_code)] // TEMP

mod options;

use libc::*;
use std::borrow::BorrowMut;
use std::cell::RefCell;
use std::collections::{BTreeMap, HashSet};
use url::Url;

thread_local!{
	static STATE: RefCell<State> = RefCell::new(State::default())
}

#[derive(Default)]
pub struct State {
	pub configs: Configs,
	pub board_configs: BoardConfigs,
	pub options: options::Options,
	pub page: Page,
	pub seen_posts: HashSet<u64>,
	pub seen_replies: HashSet<u64>,
	pub mine: HashSet<u64>,
	pub hidden: HashSet<u64>,
}

// Server-wide global configurations
#[derive(Deserialize, Default)]
#[allow(non_snake_case)]
pub struct Configs {
	captcha: bool,
	mature: bool,
	disableUserBoards: bool,
	pruneThreads: bool,
	threadExpiryMin: u32,
	threadExpiryMax: u32,
	defaultLang: String,
	defaultCSS: String,
	imageRootOverride: String,
	links: BTreeMap<String, String>,
}

// Board-specific configurations
#[derive(Deserialize, Default)]
#[allow(non_snake_case)]
pub struct BoardConfigs {
	readOnly: bool,
	textOnly: bool,
	forcedAnon: bool,
	title: String,
	notice: String,
	rules: String,
}

// Describes the current page
#[derive(Default)]
pub struct Page {
	catalog: bool,
	thread: u64,
	last_n: u32,
	page: u32,
	board: String,
	href: String,
}

impl Page {
	// Parse page URL
	pub fn from_url(url: &str) -> Page {
		let u = Url::parse(url).unwrap();
		let mut path = u.path_segments().unwrap();
		let board = path.next().unwrap();
		let (thread, catalog): (u64, bool) = match path.next() {
			Some(s) => {
				if s == "catalog" {
					(0, true)
				} else {
					match s.parse::<u64>() {
						Ok(id) => (id, false),
						_ => (0, false),
					}
				}
			}
			None => (0, false),
		};

		let mut q = u.query_pairs();
		macro_rules! parse {
			($key:expr) => (
				match q.find(|q| q.0 == $key) {
					Some(q) => {
						match q.1.parse::<u32>() {
							Ok(i) => i,
							_ => 0,
						}
					}
					None => 0,
				}
			)
		}

		Page {
			thread,
			catalog,
			page: parse!("page"),
			last_n: parse!("last"),
			board: board.to_string(),
			href: url.to_string(),
		}
	}
}

pub fn load() {
	with_state(|state| {
		state.options = options::load();
		state.page = Page::from_url(&from_C_string!(page_url()));
	})
}

// Run function, with the state of the application as an argument
pub fn with_state<F, R>(func: F) -> R
where
	F: FnOnce(&mut State) -> R,
{
	STATE.with(|r| func(r.borrow_mut().borrow_mut()))
}

extern "C" {
	fn page_url() -> *mut c_char;
}
