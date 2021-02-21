use crate::message::Message;
use actix::prelude::*;
use common::MessageType;
use serde::Serialize;
use std::{
	ops::{Deref, DerefMut},
	sync::{Arc, Mutex},
};

/// Atomically generates unique sequential u64 IDs
#[derive(Default, Clone)]
pub struct IDGenerator {
	counter: Arc<Mutex<u64>>,
}

impl IDGenerator {
	/// Return the next unique ID
	#[inline]
	pub fn next(&self) -> u64 {
		let mut ptr = self.counter.lock().unwrap();
		*ptr += 1;
		*ptr
	}
}

/// Boxed error shorthand
pub type Err = Box<dyn std::error::Error + Send + Sync>;

/// Boxed error result type shorthand
pub type DynResult<T = ()> = Result<T, Err>;

/// Return a string as error
#[macro_export]
macro_rules! str_err {
	($msg:expr) => {
		return Err($msg.to_owned().into());
	};
	($fmt:expr, $( $args:tt )* ) => {
		str_err!(format!($fmt, $($args)*))
    };
}

/// Notify the Actor there are updates it should fetch and process
#[derive(Message)]
#[rtype(result = "()")]
pub struct WakeUp;

/// Schedule processing of the buffered state
#[derive(Message)]
#[rtype(result = "()")]
pub struct Pulse;

/// Wrapper for caching an encoded message generated from T.
///
/// Mutably dereferencing MessageCacher clears the cached message
//
// TODO: bind to specific MessageType, when const generics stabilize,
#[derive(Debug, Default)]
pub struct MessageCacher<T: Serialize> {
	val: T,
	cached: Option<(MessageType, Message)>,
}

impl<T: Serialize> MessageCacher<T> {
	/// Create a new wrapped value
	#[inline]
	pub fn new(val: T) -> MessageCacher<T> {
		MessageCacher::<T> { val, cached: None }
	}

	/// Return the encoded message generated from the value
	pub fn get_message(
		&mut self,
		typ: common::MessageType,
	) -> std::io::Result<Message> {
		Ok(match &mut self.cached {
			Some((t, m)) if &typ == t => m.clone(),
			_ => {
				let msg =
					Message::from(common::Encoder::encode(typ, &self.val)?);
				self.cached = Some((typ, msg.clone()));
				msg
			}
		})
	}

	/// Consume message cacher and return inner value
	#[inline]
	pub fn get(self) -> T {
		self.val
	}
}

impl<T: Serialize> Deref for MessageCacher<T> {
	type Target = T;

	#[inline]
	fn deref(&self) -> &Self::Target {
		&self.val
	}
}

impl<T: Serialize> AsRef<T> for MessageCacher<T> {
	#[inline]
	fn as_ref(&self) -> &T {
		&self.val
	}
}

impl<T: Serialize> DerefMut for MessageCacher<T> {
	#[inline]
	fn deref_mut(&mut self) -> &mut Self::Target {
		self.cached = None;
		&mut self.val
	}
}

impl<T: Serialize> From<T> for MessageCacher<T> {
	#[inline]
	fn from(v: T) -> MessageCacher<T> {
		Self::new(v)
	}
}

/// Return current Unix timestamp
pub fn now() -> u32 {
	std::time::SystemTime::now()
		.duration_since(std::time::UNIX_EPOCH)
		.unwrap()
		.as_secs() as u32
}

/// Versioned copy on write snapshotable container.
///
/// Must not be modified concurrently to maintain version consistency.
#[derive(Debug, Default)]
pub struct SnapshotSource<T: Clone + Sync> {
	/// Mutable data for snapshot generation
	val: T,

	/// ID for tracking snapshot versions
	snapshot_version: u64,

	/// Snapshot generated for the current iteration of src, if any
	last_snapshot: Option<Snapshot<T>>,
}

impl<T: Clone + Sync> SnapshotSource<T> {
	/// Create new SnapshotSource initialized with val
	#[inline]
	pub fn new(val: T) -> Self {
		Self {
			val,
			snapshot_version: 0,
			last_snapshot: None,
		}
	}

	/// Create new snapshot of the current state of SnapshotSource
	pub fn snapshot(&mut self) -> Snapshot<T> {
		// Crate a new snapshot lazily to avoid copying on mutations between
		// snapshots
		match &self.last_snapshot {
			Some(s) => s.clone(),
			None => {
				self.snapshot_version += 1;
				let s = Snapshot {
					snapshot_version: self.snapshot_version,
					val: Arc::new(self.val.clone()),
				};
				self.last_snapshot = Some(s.clone());
				s
			}
		}
	}
}

impl<T: Clone + Sync> From<T> for SnapshotSource<T> {
	#[inline]
	fn from(val: T) -> Self {
		Self::new(val)
	}
}

impl<T: Clone + Sync> PartialEq<Snapshot<T>> for SnapshotSource<T> {
	#[inline]
	fn eq(&self, other: &Snapshot<T>) -> bool {
		self.last_snapshot.is_some()
			&& self.snapshot_version == other.snapshot_version
	}
}

impl<T: Clone + Sync> Deref for SnapshotSource<T> {
	type Target = T;

	#[inline]
	fn deref(&self) -> &T {
		&self.val
	}
}

impl<T: Clone + Sync> DerefMut for SnapshotSource<T> {
	#[inline]
	fn deref_mut(&mut self) -> &mut T {
		// Invalidate any saved snapshots because val might be modified
		self.last_snapshot = None;
		&mut self.val
	}
}

/// Immutable snapshot of SnapshotSource
#[derive(Debug, Clone, Default)]
pub struct Snapshot<T: Clone + Sync> {
	/// Snapshot version for cheap comparison
	snapshot_version: u64,

	/// Contained value
	val: Arc<T>,
}

impl<T: Clone + Sync> actix::Message for Snapshot<T> {
	type Result = ();
}

impl<T: Clone + Sync> PartialEq<Snapshot<T>> for Snapshot<T> {
	#[inline]
	fn eq(&self, other: &Snapshot<T>) -> bool {
		self.snapshot_version == other.snapshot_version
	}
}

impl<T: Clone + Sync> Eq for Snapshot<T> {}

impl<T: Clone + Sync> Deref for Snapshot<T> {
	type Target = T;

	#[inline]
	fn deref(&self) -> &T {
		&self.val
	}
}

/// Run function in the Rayon thread pool and return its result.
pub async fn run_in_rayon<F, R>(f: F) -> DynResult<R>
where
	F: FnOnce() -> R + Send + 'static,
	R: Send + 'static,
{
	let (send, receive) = tokio::sync::oneshot::channel();
	rayon::spawn(move || {
		// Ignore failure to receive. The parent future might have been dropped.
		std::mem::drop(send.send(f()));
	});
	receive.await.map_err(|e| e.to_string().into())
}
