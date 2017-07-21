#![allow(dead_code)] // TEMP

mod options;

use libc::*;
use std::borrow::BorrowMut;
use std::cell::RefCell;
use std::collections::{BTreeMap, HashSet};

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
}

impl Page {
	// Parse page URL
	pub fn from_url(path: &str, query: &str) -> Page {
		let mut path_split = path[1..].split("/");
		let board = path_split.next().unwrap();
		println!("{}", board);
		let (thread, catalog): (u64, bool) = match path_split.next() {
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

		let mut page = 0u32;
		let mut last_n = 0u32;
		if query != "" {
			let mut split = query[1..].split("&");
			let mut parse =
				|key: &str| match split.find(|q| q.starts_with(key)) {
					Some(q) => {
						match q.split("=").last() {
							Some(i) => {
								match i.parse::<u32>() {
									Ok(i) => i,
									_ => 0,
								}
							}
							None => 0,
						}
					}
					None => 0,
				};

			page = parse("page");
			last_n = parse("last");
		}

		Page {
			thread,
			catalog,
			page,
			last_n,
			board: board.to_string(),
		}
	}
}

pub fn load() {
	with_state(|state| {
		state.options = options::load();
		state.page = Page::from_url(
			&from_C_string!(page_path()),
			&from_C_string!(page_query()),
		);
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
	fn page_path() -> *mut c_char;
	fn page_query() -> *mut c_char;
}
