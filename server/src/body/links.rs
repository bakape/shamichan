use crate::{db, util};
use common::payloads::post_body::{Node, PendingNode, PostLink};
use std::{collections::HashMap, sync::Arc};
use tokio::sync::RwLock as AsyncRWLock;

#[derive(Clone)]
pub enum PostLocation {
	NotFetched,
	DoesNotExist,
	Exists { thread: u64, page: u32 },
}

impl Default for PostLocation {
	#[inline]
	fn default() -> Self {
		Self::NotFetched
	}
}

type PostLocationCache = HashMap<u64, Arc<AsyncRWLock<PostLocation>>>;
static __ONCE: std::sync::Once = std::sync::Once::new();
static mut __GLOBAL: Option<std::sync::RwLock<PostLocationCache>> = None;

#[inline]
#[cold]
fn __init() {
	__ONCE.call_once(|| {
		unsafe { __GLOBAL = Some(Default::default()) };
	});
}

/// Open post location cache for reading
#[inline]
fn read_cache<F, R>(cb: F) -> R
where
	F: FnOnce(&PostLocationCache) -> R,
{
	__init();
	cb(&*unsafe { __GLOBAL.as_ref().unwrap().read().unwrap() })
}

/// Open post location cache for writing
#[inline]
fn write_cache<F, R>(cb: F) -> R
where
	F: FnOnce(&mut PostLocationCache) -> R,
{
	__init();
	cb(&mut *unsafe { __GLOBAL.as_ref().unwrap().write().unwrap() })
}

/// Fetch post location from the DB or cache
pub async fn post_location(id: u64) -> util::DynResult<PostLocation> {
	use PostLocation::*;

	let rec = write_cache(|c| c.entry(id).or_default().clone());
	match &*rec.read().await {
		l @ Exists { .. } | l @ DoesNotExist => return Ok(l.clone()),
		NotFetched => (),
	};

	let mut rec = rec.write().await;
	Ok(match &mut *rec {
		// Race with another thread
		l @ Exists { .. } | l @ DoesNotExist => l.clone(),

		// Perform fetch
		l @ NotFetched => {
			let loc = match db::get_post_parenthood(id).await? {
				Some((thread, page)) => PostLocation::Exists { thread, page },
				None => DoesNotExist,
			};
			*l = loc.clone();
			loc
		}
	})
}

/// Insert a post location into the cache
pub fn cache_location(id: u64, thread: u64, page: u32) {
	let loc = Arc::new(AsyncRWLock::new(PostLocation::Exists { thread, page }));
	write_cache(|c| c.insert(id, loc));
}

/// Parse post links and configured references
pub fn parse_link(word: &str, flags: u8) -> Option<Node> {
	if !word.starts_with(">>") {
		return None;
	}

	let prepend_extra_gt = |n: Node, extra_gt: usize| -> Node {
		if extra_gt > 0 {
			Node::Siblings([Node::Text(">".repeat(extra_gt)).into(), n.into()])
		} else {
			n
		}
	};

	let parse_post_link = |extra_gt: usize| -> Option<Node> {
		use PostLocation::*;

		word[2 + extra_gt as usize..]
			.parse()
			.ok()
			.map(|id| {
				match match read_cache(|c| c.get(&id).cloned()) {
					Some(m) => m.try_read().map(|m| m.clone()),
					None => Ok(NotFetched),
				} {
					Ok(DoesNotExist) => None,
					Ok(Exists { page, thread }) => {
						Some(Node::PostLink(PostLink {
							id,
							thread: thread,
							page: page,
						}))
					}
					// Error is always tokio::sync::TryLockError - failure to
					// lock
					Ok(NotFetched) | Err(_) => {
						if flags & super::OPEN != 0 {
							// Will need to fetch later
							Some(Node::Pending(PendingNode::PostLink(id)))
						} else {
							// Just keep as text for now
							None
						}
					}
				}
			})
			.flatten()
			.map(|n| prepend_extra_gt(n, extra_gt))
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

		let id = &word[start..end];
		crate::config::get().public.links.get(id).map(|url| {
			prepend_extra_gt(
				Node::Reference {
					label: id.into(),
					url: url.into(),
				},
				extra_gt,
			)
		})
	};

	let mut extra_gt = 0;
	for c in word.chars().skip(2) {
		match c {
			'>' => {
				extra_gt += 1;
			}
			'0'..='9' => return parse_post_link(extra_gt),
			'/' => return parse_reference(extra_gt),
			_ => return None,
		}
	}
	None
}
