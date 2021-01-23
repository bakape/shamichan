use crate::{
	message::Message,
	util::{DynResult, MessageCacher},
};
use actix_web::web::Bytes;
use common::{
	payloads::{ImmutablePage, Post},
	Encoder, MessageType,
};
use rayon::prelude::*;
use std::{
	collections::HashMap,
	ops::{Deref, DerefMut},
};

/// Wraps a mutable page's posts and caches the resulting fetch message
#[derive(Debug, Default)]
pub struct MutablePage {
	cache: Option<Message>,
	posts: HashMap<u64, MessageCacher<Post>>,
}

impl MutablePage {
	fn new(posts: HashMap<u64, MessageCacher<Post>>) -> Self {
		Self { posts, cache: None }
	}

	/// Retrieve a cached message or generate a new one
	pub fn get_message(&mut self) -> std::io::Result<Message> {
		macro_rules! static_encode {
			($name:ident, $type:ident) => {
				lazy_static::lazy_static! {
					static ref $name: Message =
						Encoder::encode(
							MessageType::$type,
							&(),
						)
						.unwrap()
						.into();
				}
			};
		}
		static_encode! {START, PartitionedPageStart}
		static_encode! {END, PartitionedPageEnd}

		Ok(match &mut self.cache {
			Some(m) => m.clone(),
			None => {
				let mut parts =
					Vec::<Message>::with_capacity(self.posts.len() + 2);
				parts.push(START.clone());
				parts.extend(
					self.posts
						.par_iter_mut()
						.map(|(_, p)| p.get_message(MessageType::Post))
						.collect::<std::io::Result<Vec<_>>>()?,
				);
				parts.push(END.clone());

				let joined = Message::new(Encoder::join(parts));
				self.cache = joined.clone().into();
				joined
			}
		})
	}

	/// Retrieve a cached message, if any
	pub fn get_cached_message(&self) -> Option<Message> {
		self.cache.clone()
	}
}

impl Deref for MutablePage {
	type Target = HashMap<u64, MessageCacher<Post>>;

	fn deref(&self) -> &Self::Target {
		&self.posts
	}
}

impl DerefMut for MutablePage {
	fn deref_mut(&mut self) -> &mut Self::Target {
		self.cache = None;
		&mut self.posts
	}
}

/// Contains thread page data
#[derive(Debug)]
pub enum PageRecord {
	/// Page exists in the database but has not been fetched yet
	Unfetched,

	/// Contains open posts that can still be edited or is bellow the
	/// page capacity of 100
	Mutable(MutablePage),

	/// Does not contain any open posts and is at full page capacity
	Immutable(Message),
}

impl Default for PageRecord {
	fn default() -> Self {
		Self::Unfetched
	}
}

impl PageRecord {
	/// Construct new mutable PageRecord
	pub fn new_mutable(posts: impl IntoIterator<Item = Post>) -> Self {
		Self::Mutable(MutablePage::new(
			posts.into_iter().map(|p| (p.id, p.into())).collect(),
		))
	}

	/// Construct new immutable PageRecord
	pub async fn new_immutable(page: &ImmutablePage) -> DynResult<Self> {
		let buf = Encoder::encode(MessageType::Page, page)?;
		let m = Box::new(
			actix_web::web::block(move || -> DynResult<memmap::Mmap> {
				let mut m =
					memmap::MmapOptions::new().len(buf.len()).map_anon()?;
				m.clone_from_slice(&buf);
				Ok(m.make_read_only()?)
			})
			.await?,
		);

		// XXX: This leaks virtual memory, if the thread is deleted.
		// To not leak the mmap you'd need to extend Bytes with construction
		// from an owned AsRef<[u8]> (AsRef<[u8]> + Drop ?) or just MMap,
		// and have it drop the memory map, when Bytes is dropped.
		Ok(Self::Immutable(Bytes::from_static(Box::leak(m)).into()))
	}

	/// Returns if a page can be considered immutable
	pub fn can_be_made_immutable<'a>(
		mut posts: impl ExactSizeIterator<Item = &'a Post>,
	) -> bool {
		posts.len() == 100 && posts.all(|p| !p.open)
	}
}
