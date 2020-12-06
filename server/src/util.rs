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
	/// Create new IDGenerator starting count form 1
	pub fn new() -> Self {
		Default::default()
	}

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
#[derive(Debug)]
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
}

impl<T: Serialize> Deref for MessageCacher<T> {
	type Target = T;

	fn deref(&self) -> &Self::Target {
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
