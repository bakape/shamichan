use super::{Result, OPEN};
use protocol::payloads::post_body::{Node, PostLink};
use std::{
	collections::HashMap,
	sync::{Arc, RwLock, TryLockError},
};

#[derive(Clone)]
pub enum PostLocation {
	NotFetched,
	DoesNotExist,
	Exists { thread: u64, page: u32 },
}

impl Default for PostLocation {
	fn default() -> Self {
		Self::NotFetched
	}
}

// Can't pass generic types to the macro
type PostLocationCache = HashMap<u64, Arc<RwLock<PostLocation>>>;

// Cache of post locations for post links
protocol::gen_global! {
	,
	__read_cache, // Not used
	,
	with_location_cache,
	PostLocationCache
}

// Read post location from cache or DB.
//
// fetch: fetch location from DB, if not yet in cache
pub fn post_location(id: u64, fetch: bool) -> Result<PostLocation> {
	use PostLocation::*;

	let store = with_location_cache(|c| c.entry(id).or_default().clone());
	let read_loc = || -> Result<PostLocation> {
		Ok(store.read().map_err(|e| e.to_string())?.clone())
	};

	let loc = read_loc()?;
	Ok(match loc {
		// Dedup concurrent DB fetches with write lock
		NotFetched if fetch => match store.try_write() {
			Ok(mut store) => {
				let loc = match crate::bindings::get_post_parenthood(id) {
					Ok(Some((thread, page))) => Exists { thread, page },
					Ok(None) => DoesNotExist,
					Err(e) => Err(e)?,
				};
				*store = loc.clone();
				loc
			}
			Err(TryLockError::Poisoned(e)) => Err(e.to_string())?,
			Err(TryLockError::WouldBlock) => {
				let loc = read_loc()?;
				match loc {
					DoesNotExist | Exists { .. } => loc,
					NotFetched => Err("concurrent lookup failed".to_owned())?,
				}
			}
		},
		_ => loc,
	})
}

// Insert a post location into the cache
pub fn cache_location(id: u64, thread: u64, page: u32) {
	with_location_cache(|c| {
		c.insert(
			id,
			Arc::new(RwLock::new(PostLocation::Exists { thread, page })),
		);
	});
}

// Parse post links and configured references
pub fn parse_link(word: &str, flags: u8) -> Result<Option<Node>> {
	if !word.starts_with(">>") {
		return Ok(None);
	}

	let prepend_extra_gt = |n: Node, extra_gt: usize| -> Node {
		if extra_gt > 0 {
			Node::Siblings([Node::Text(">".repeat(extra_gt)).into(), n.into()])
		} else {
			n
		}
	};

	let parse_post_link = |extra_gt: usize| -> Result<Option<Node>> {
		use PostLocation::*;

		Ok(match word[2 + extra_gt as usize..].parse() {
			Ok(id) => match post_location(id, flags & OPEN == 0)? {
				DoesNotExist => None,
				NotFetched => Some(Node::PostLink(PostLink {
					id,
					thread: 0,
					page: 0,
				})),
				Exists { page, thread } => {
					Some(Node::PostLink(PostLink { id, thread, page }))
				}
			},
			_ => None,
		}
		.map(|n| prepend_extra_gt(n, extra_gt)))
	};

	let parse_reference = |mut extra_gt: usize| -> Option<Node> {
		if extra_gt == 0 {
			return None;
		}
		extra_gt -= 1;

		#[rustfmt::skip]
		macro_rules! slash_pos {
			($s:expr) => {
				match $s.find('/') {
					Some(i) => i,
					None => return None,
				}
			};
		}

		let start = slash_pos!(word) + 1;
		let end = slash_pos!(word[start..]);

		// Check this before requesting a lock on the configs to reduce
		// contention
		if end - start <= 1 || end != word.len() - 1 {
			return None;
		}

		crate::config::read(|c| {
			let id = &word[start..end];
			c.links.get(id).map(|url| {
				prepend_extra_gt(
					Node::Reference {
						label: id.into(),
						url: url.into(),
					},
					extra_gt,
				)
			})
		})
	};

	let mut extra_gt = 0;
	for c in word.chars().skip(2) {
		match c {
			'>' => {
				extra_gt += 1;
			}
			'0'..='9' => return parse_post_link(extra_gt),
			'/' => return Ok(parse_reference(extra_gt)),
			_ => return Ok(None),
		}
	}
	Ok(None)
}
