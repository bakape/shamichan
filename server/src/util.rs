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
	pub fn get(self) -> T {
		self.val
	}
}

impl<T: Serialize> Deref for MessageCacher<T> {
	type Target = T;

	fn deref(&self) -> &Self::Target {
		&self.val
	}
}

impl<T: Serialize> AsRef<T> for MessageCacher<T> {
	fn as_ref(&self) -> &T {
		&self.val
	}
}

impl<T: Serialize> DerefMut for MessageCacher<T> {
	fn deref_mut(&mut self) -> &mut Self::Target {
		self.cached = None;
		&mut self.val
	}
}

impl<T: Serialize> From<T> for MessageCacher<T> {
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
pub struct SnapshotSource<T: Clone + Sync>(Snapshot<T>);

impl<T: Clone + Sync> SnapshotSource<T> {
	/// Create new SnapshotSource initialized with val
	pub fn new(val: T) -> Self {
		Self(Snapshot::<T> {
			// Start at 1 because 0 is the default state
			update_counter: 1,
			value: val.into(),
		})
	}

	/// Create new snapshot of the current state of SnapshotSource
	pub fn snapshot(&self) -> Snapshot<T> {
		self.0.clone()
	}

	/// Set new value of SnapshotSource
	pub fn set(&mut self, val: T) {
		self.0.update_counter += 1;
		self.0.value = val.into();
	}

	/// Modify the current value of SnapshotSource
	pub fn modify(&mut self, f: impl FnOnce(&mut T)) {
		let mut v = (*self.0.value).clone();
		f(&mut v);
		self.set(v);
	}
}

impl<T: Clone + Sync> From<T> for SnapshotSource<T> {
	fn from(val: T) -> Self {
		Self::new(val)
	}
}

impl<T: Clone + Sync> PartialEq<Snapshot<T>> for SnapshotSource<T> {
	fn eq(&self, other: &Snapshot<T>) -> bool {
		&self.0 == other
	}
}

impl<T: Clone + Sync> Deref for SnapshotSource<T> {
	type Target = T;

	fn deref(&self) -> &T {
		&self.0
	}
}

/// Immutable snapshot of SnapshotSource
#[derive(Debug, Clone, Default)]
pub struct Snapshot<T: Clone + Sync> {
	/// Incremented on any modification to clients for cheap comparison
	update_counter: usize,

	/// Contained value
	value: Arc<T>,
}

impl<T: Clone + Sync> actix::Message for Snapshot<T> {
	type Result = ();
}

impl<T: Clone + Sync> PartialEq<Snapshot<T>> for Snapshot<T> {
	fn eq(&self, other: &Snapshot<T>) -> bool {
		self.update_counter == other.update_counter
	}
}

impl<T: Clone + Sync> Eq for Snapshot<T> {}

impl<T: Clone + Sync> Deref for Snapshot<T> {
	type Target = T;

	fn deref(&self) -> &T {
		&self.value
	}
}

/// Run function in the Rayon thread pool and return its result
pub async fn run_in_rayon<F, R>(f: F) -> DynResult<R>
where
	F: FnOnce() -> R + Send + 'static,
	R: Send + 'static,
{
	let (send, receive) = tokio::sync::oneshot::channel();
	rayon::spawn(move || {
		std::mem::drop(send.send(f()));
	});
	receive.await.map_err(|e| e.to_string().into())
}
