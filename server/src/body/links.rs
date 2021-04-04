use crate::{db, util};
use common::payloads::{
	post_body::{Node, PendingNode},
	Post,
};
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

/// Known post location that can be inserted into the cache
pub struct KnownPostLocation {
	pub id: u64,
	pub thread: u64,
	pub page: u32,
}

impl From<&Post> for KnownPostLocation {
	fn from(p: &Post) -> Self {
		Self {
			id: p.id,
			thread: p.thread,
			page: p.page,
		}
	}
}

/// Register a post as not existing. Only used in tests.
#[cfg(test)]
pub fn register_non_existent_post(id: u64) {
	write_cache(|c| {
		c.insert(id, Arc::new(AsyncRWLock::new(PostLocation::DoesNotExist)));
	});
}

/// Insert known post locations into the cache
pub fn cache_locations<T>(it: impl Iterator<Item = T>)
where
	T: Into<KnownPostLocation>,
{
	// Collect into vector ahead of time for les cache lock contention
	let ex = it
		.map(|t| {
			let loc = t.into();
			(
				loc.id,
				Arc::new(AsyncRWLock::new(PostLocation::Exists {
					thread: loc.thread,
					page: loc.page,
				})),
			)
		})
		.collect::<Vec<_>>();
	write_cache(|c| c.extend(ex));
}

/// Parses a potential post link and return the target post's location
fn parse_post_link(word: &str, extra_gt: usize) -> Option<(u64, PostLocation)> {
	word[2 + extra_gt as usize..].parse().ok().map(|id| {
		match read_cache(|c| c.get(&id).cloned()) {
			Some(m) => m
				.try_read()
				.map(|m| (id, m.clone()))
				// Error is always tokio::sync::TryLockError - failure to
				// lock
				.unwrap_or((id, PostLocation::NotFetched)),
			None => (id, PostLocation::NotFetched),
		}
	})
}

/// Parse a configured reference to some URI.
/// This does not check the server configurations to validate - only the syntax.
fn parse_reference<'a>(word: &'a str, extra_gt: &mut usize) -> Option<&'a str> {
	if *extra_gt == 0 {
		return None;
	}

	macro_rules! slash_pos {
		($s:expr) => {
			match $s.bytes().position(|b| b == b'/') {
				Some(i) => i,
				None => return None,
			}
		};
	}

	let start = slash_pos!(word) + 1;
	let end = slash_pos!(word[start..]) + start;

	// Check this before requesting a lock on the configs to reduce
	// contention
	if end - start <= 1 || end != word.len() - 1 {
		None
	} else {
		*extra_gt -= 1;
		Some(&word[start..end])
	}
}

/// Parse links and run handlers on any matches.
/// The callbacks also take the number of extra preceding `>` signs as the last
/// argument
fn parse_links_inner<R>(
	word: &str,
	on_post_link_match: impl FnOnce(u64, PostLocation, usize) -> Option<R>,
	on_reference_match: impl FnOnce(&str, usize) -> Option<R>,
) -> Option<R> {
	if !word.starts_with(">>") {
		return None;
	}

	let mut extra_gt = 0;
	for c in word.bytes().skip(2) {
		match c {
			b'>' => {
				extra_gt += 1;
			}
			b'0'..=b'9' => {
				return parse_post_link(word, extra_gt)
					.map(|(id, loc)| on_post_link_match(id, loc, extra_gt))
					.flatten()
			}
			b'/' => {
				return parse_reference(word, &mut extra_gt)
					.map(|id| on_reference_match(id, extra_gt))
					.flatten()
			}
			_ => return None,
		}
	}
	None
}

/// Parse post links and configured references.
///
/// Returns, if a valid link has been parsed and written to dst.
pub fn parse_link(mut dst: &mut Node, word: &str) -> bool {
	match parse_links_inner(
		word,
		|id, loc, extra_gt| {
			use PostLocation::*;

			match loc {
				DoesNotExist => None,
				Exists { page, thread } => {
					Some(Node::PostLink { id, thread, page })
				}
				NotFetched => Some(Node::Pending(PendingNode::PostLink(id))),
			}
			.map(|n| (extra_gt, n))
		},
		|id, extra_gt| {
			crate::config::get().public.links.get(id).map(|url| {
				(
					extra_gt,
					Node::Reference {
						label: id.into(),
						url: url.into(),
					},
				)
			})
		},
	) {
		Some((extra_gt, n)) => {
			if extra_gt > 0 {
				dst += ">".repeat(extra_gt);
			}
			dst += n;
			true
		}
		None => false,
	}
}

/// Detect any post links and configured references and return quotation level,
/// if any matched
pub fn detect_link(word: &str) -> Option<usize> {
	parse_links_inner(
		word,
		|_, loc, extra_gt| {
			use PostLocation::*;

			match loc {
				DoesNotExist => None,
				Exists { .. } | NotFetched => Some(extra_gt),
			}
		},
		|id, extra_gt| {
			if crate::config::get().public.links.contains_key(id) {
				Some(extra_gt)
			} else {
				None
			}
		},
	)
}
