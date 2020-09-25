use crate::util;
use serde::{Deserialize, Serialize};

/// Identifies a global index or thread feed
#[derive(Serialize, Deserialize, Debug, PartialEq, Eq, Clone)]
pub enum FeedID {
	Unset,
	Index,
	Catalog,
	Thread { id: u64, page: i32 },
}

impl Default for FeedID {
	fn default() -> FeedID {
		FeedID::Unset
	}
}

impl FeedID {
	/// Return corresponding integer feed code used by the server
	pub fn as_u64(&self) -> u64 {
		use FeedID::*;

		match self {
			// Should never match anything unless this instance runs for
			// decades unchanged
			Unset => std::u64::MAX,
			Index | Catalog => 0,
			Thread { id, .. } => *id,
		}
	}
}

/// Post or page margin to scroll to
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

/// Location the app can navigate to
#[derive(Default, Serialize, Deserialize, Clone, Eq, PartialEq, Debug)]
pub struct Location {
	pub feed: FeedID,

	/// Focus a post after a successful render
	pub focus: Option<Focus>,
}

impl Location {
	pub fn from_path() -> Location {
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

	pub fn path(&self) -> String {
		use FeedID::*;
		use Focus::*;

		let mut w: String = match &self.feed {
			Unset | Index => "/".into(),
			Catalog => "/catalog".into(),
			Thread { id, page } => format!("/threads/{}/{}", id, page),
		};
		if let Some(f) = &self.focus {
			match f {
				Bottom => {
					w += "#bottom";
				}
				Top => {
					w += "#top";
				}
				Post(id) => {
					use std::fmt::Write;

					write!(w, "#p-{}", id).unwrap();
				}
			}
		}
		w
	}

	/// Returns, if this is a single thread page
	pub fn is_thread(&self) -> bool {
		matches!(self.feed, FeedID::Thread { .. })
	}
}
