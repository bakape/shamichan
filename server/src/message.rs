use bytes::Bytes;
use common::Decoder;

/// Reusable message buffer wrapper with AsRef[u8]
#[derive(Clone)]
pub struct Message(Bytes);

impl Message {
	pub fn new(buf: impl Into<Bytes>) -> Self {
		Self(buf.into())
	}
}

impl AsRef<[u8]> for Message {
	fn as_ref(&self) -> &[u8] {
		self.0.as_ref()
	}
}

impl From<Bytes> for Message {
	fn from(v: Bytes) -> Self {
		Self(v)
	}
}

impl From<Vec<u8>> for Message {
	fn from(v: Vec<u8>) -> Self {
		Self::new(v)
	}
}

impl Into<Bytes> for Message {
	fn into(self) -> Bytes {
		self.0
	}
}

impl std::fmt::Debug for Message {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		let d = match Decoder::new(self.as_ref()) {
			Ok(d) => d,
			Err(e) => return write!(f, "Message (failed to decode: {})", e),
		};

		write!(f, "Message [")?;
		for (i, t) in d.all_types().iter().enumerate() {
			if i != 0 {
				write!(f, ", ")?;
			}
			write!(f, "{:?}", t)?;
		}
		write!(f, "]")
	}
}

/// Messages to be sent to a specific client
#[derive(Debug)]
pub struct ClientMessage {
	pub client: u64,
	pub msg: Message,
}

/// Used for aggregation of messages in parallel
#[derive(Default, Debug)]
pub struct MessageSet {
	/// Messages to be sent on the global thread index feed
	pub global_feed_messages: Vec<Message>,

	/// Messages to be sent to specific clients on specific threads
	pub thread_messages: Vec<ClientMessage>,
}

impl MessageSet {
	#[allow(unused)]
	pub fn is_empty(&self) -> bool {
		self.global_feed_messages.is_empty() && self.thread_messages.is_empty()
	}
}
