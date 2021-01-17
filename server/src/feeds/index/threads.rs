use crate::{message::Message, util::MessageCacher};
use common::{payloads::ThreadWithPosts, Encoder, MessageType};
use rayon::prelude::*;
use std::{
	collections::{HashMap, HashSet},
	ops::{Deref, DerefMut},
};

/// Wraps a thread index's threads and caches the resulting fetch message
#[derive(Debug, Default)]
pub struct Threads {
	cache: Option<Message>,
	used_tags: Option<Message>,
	threads: HashMap<u64, MessageCacher<ThreadWithPosts>>,
}

impl Threads {
	pub fn new(threads: HashMap<u64, MessageCacher<ThreadWithPosts>>) -> Self {
		Self {
			threads,
			cache: None,
			used_tags: None,
		}
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
		static_encode! {START, PartitionedThreadIndexStart}
		static_encode! {END, PartitionedThreadIndexEnd}

		Ok(match &mut self.cache {
			Some(m) => m.clone(),
			None => {
				let mut parts =
					Vec::<Message>::with_capacity(self.threads.len() + 2);
				parts.push(START.clone());
				parts.extend(
					self.threads
						.par_iter_mut()
						.map(|(_, t)| {
							t.get_message(MessageType::ThreadAbbreviated)
						})
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

	/// Return set of used tags across all active threads
	pub fn used_tags(&mut self) -> std::io::Result<Message> {
		Ok(match &self.used_tags {
			Some(m) => m.clone(),
			None => {
				let mut tags = self
					.threads
					.values()
					.flat_map(|t| t.thread_data.tags.iter())
					.cloned()
					.collect::<HashSet<String>>()
					.into_iter()
					.collect::<Vec<String>>();
				tags.sort_unstable();
				let msg = Message::new(Encoder::encode(
					MessageType::UsedTags,
					&tags,
				)?);
				self.used_tags = msg.clone().into();
				msg
			}
		})
	}
}

impl Deref for Threads {
	type Target = HashMap<u64, MessageCacher<ThreadWithPosts>>;

	fn deref(&self) -> &Self::Target {
		&self.threads
	}
}

impl DerefMut for Threads {
	fn deref_mut(&mut self) -> &mut Self::Target {
		self.cache = None;
		self.used_tags = None;
		&mut self.threads
	}
}
