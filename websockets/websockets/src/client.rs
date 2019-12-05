use super::common::DynResult;
use protocol::AuthKey;
use protocol::*;
use std::net::IpAddr;

// Maps to a websocket client on the Go side
pub struct Client {
	// ID of client used in various registries
	id: u64,

	// IP of client connection
	ip: IpAddr,

	// Used to authenticate the client
	key: Option<AuthKey>,
}

// Return a string as error
macro_rules! str_err {
	($msg:expr) => {
		return Err($msg.into());
	};
	($fmt:expr, $( $args:tt )* ) => {
		str_err!(format!($fmt, $($args)*))
    };
}

impl Client {
	// Create fresh unconnected client
	pub fn new(id: u64, ip: IpAddr) -> Self {
		Self {
			id: id,
			ip: ip,
			key: None,
		}
	}

	// Handle received message
	pub fn receive_message(&mut self, buf: &[u8]) -> DynResult {
		let mut dec = Decoder::new(buf)?;
		let mut typ = match dec.peek_type() {
			Some(t) => t,
			None => str_err!("empty message received"),
		};

		if self.key.is_none() {
			if typ != MessageType::Handshake {
				str_err!("first message must be handshake");
			}
			let msg: Handshake = dec.read_next()?;
			if msg.protocol_version != VERSION {
				str_err!("protocol version mismatch: {}", msg.protocol_version);
			}
			super::registry::set_client_key(self.id, &msg.key);
			self.key = Some(msg.key);

			if dec.peek_type() != Some(MessageType::Synchronize) {
				str_err!("second message in first batch must be sync request");
			}
			self.synchronize(&mut dec)?;
		}

		unimplemented!("other message batch processing")
	}

	// Synchronize to a specific thread or board index
	fn synchronize(&mut self, dec: &mut Decoder) -> DynResult {
		let msg: SyncRequest = dec.read_next()?;

		// TODO: Check thread exists against DB.
		// bool ws_thread_exists(uint64_t id, char** err)

		super::registry::set_client_thread(self.id, msg.thread);

		unimplemented!("register + sync to thread/board feed")
	}
}
