use super::Result;
use common::payloads::post_body::{Node, PendingNode, PostLink};
use std::{
	collections::HashMap,
	sync::{Arc, RwLock},
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

/// Generate functions for safely accessing global variable behind a RWLock
#[macro_export]
macro_rules! gen_global {
	(
		$(#[$meta:meta])*
		$type:ty {
			$vis_read:vis fn $fn_read:ident();
			$vis_write:vis fn $fn_write:ident();
		}
	) => {
		static __ONCE: std::sync::Once = std::sync::Once::new();
		static mut __GLOBAL: Option<std::sync::RwLock<$type>> = None;

		fn __init() {
			__ONCE.call_once(|| {
				unsafe { __GLOBAL = Some(Default::default()) };
			});
		}

		#[allow(unused)]
		$(#[$meta])*
		$vis_read fn $fn_read<F, R>(cb: F) -> R
		where
			F: FnOnce(&$type) -> R,
		{
			__init();
			cb(&*unsafe { __GLOBAL.as_ref().unwrap().read().unwrap() })
		}

		#[allow(unused)]
		$(#[$meta])*
		$vis_write fn $fn_write<F, R>(cb: F) -> R
		where
			F: FnOnce(&mut $type) -> R,
		{
			__init();
			cb(&mut *unsafe { __GLOBAL.as_ref().unwrap().write().unwrap() })
		}
	};
}

/// Can't pass generic types to the macro
type PostLocationCache = HashMap<u64, Arc<RwLock<PostLocation>>>;

gen_global! {
	/// Cache of post locations for post links
	PostLocationCache {
		fn read();
		fn write();
	}
}

/// Read post location from cache
pub fn post_location(id: u64) -> Result<PostLocation> {
	Ok(write(|c| c.entry(id).or_default().clone())
		.read()
		.map_err(|e| e.to_string())?
		.clone())

	// TODO: async fetch function with DB lookup; must properly handle all the
	// PostLocation enum transitions

	// use PostLocation::*;

	// let store = write(|c| c.entry(id).or_default().clone());
	// let read_loc = || -> Result<PostLocation> {
	// 	Ok(store.read().map_err(|e| e.to_string())?.clone())
	// };

	// let loc = read_loc()?;
	// Ok(match loc {
	// 	// Dedup concurrent DB fetches with write lock
	// 	NotFetched if fetch => match store.try_write() {
	// 		Ok(mut store) => {
	// 			let loc = match crate::bindings::get_post_parenthood(id) {
	// 				Ok(Some((thread, page))) => Exists { thread, page },
	// 				Ok(None) => DoesNotExist,
	// 				Err(e) => Err(e)?,
	// 			};
	// 			*store = loc.clone();
	// 			loc
	// 		}
	// 		Err(TryLockError::Poisoned(e)) => Err(e.to_string())?,
	// 		Err(TryLockError::WouldBlock) => {
	// 			let loc = read_loc()?;
	// 			match loc {
	// 				DoesNotExist | Exists { .. } => loc,
	// 				NotFetched => Err("concurrent lookup failed".to_owned())?,
	// 			}
	// 		}
	// 	},
	// 	_ => loc,
	// })
}

/// Insert a post location into the cache
pub fn cache_location(id: u64, thread: u64, page: u32) {
	write(|c| {
		c.insert(
			id,
			Arc::new(RwLock::new(PostLocation::Exists { thread, page })),
		);
	});
}

/// Parse post links and configured references
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
			Ok(id) => match post_location(id)? {
				DoesNotExist => None,
				NotFetched => Some(if flags & super::OPEN != 0 {
					Node::PostLink(PostLink {
						id,
						thread: 0,
						page: 0,
					})
				} else {
					Node::Pending(PendingNode::PostLink(id))
				}),
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
			'/' => return Ok(parse_reference(extra_gt)),
			_ => return Ok(None),
		}
	}
	Ok(None)
}
