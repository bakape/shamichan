use super::common::DynResult;
use super::config;
use super::{bindings, registry, str_err};
use protocol::AuthKey;
use protocol::*;
use serde::Serialize;
use std::io;
use std::net::IpAddr;
use std::rc::Rc;

// Maps to a websocket client on the Go side
pub struct Client {
	// ID of client used in various registries
	id: u64,

	// IP of client connection
	ip: IpAddr,

	// Used to authenticate the client
	key: Option<AuthKey>,
}

macro_rules! check_len {
	// Assert collection length greater than 1 and smaller than $max
	($val:expr, $max:expr) => {
		check_len!($val, 1, $max)
	};
	// Assert collection length greater than $min and smaller than $max
	($val:expr, $min:expr, $max:expr) => {{
		let l = $val.len();
		if l < $min || l > $max {
			str_err!("invalid {} length: {}", stringify!(val), l);
			}
		}};
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
		let typ = match dec.peek_type() {
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
			registry::set_client_key(self.id, &msg.key);
			self.key = Some(msg.key);

			if dec.peek_type() != Some(MessageType::Synchronize) {
				str_err!("second message in first batch must be sync request");
			}
			self.synchronize(&mut dec)?;
		}

		loop {
			match dec.peek_type() {
				None => return Ok(()),
				Some(t) => match t {
					MessageType::CreateThread => {
						self.create_thread(&mut dec)?
					}
					_ => str_err!("unhandled message type: {:?}", t),
				},
			}
		}
	}

	// Send a private message to only this client
	fn send(&self, t: MessageType, payload: &impl Serialize) -> io::Result<()> {
		let mut enc = Encoder::new(Vec::new());
		enc.write_message(t, payload)?;
		bindings::write_message(self.id, Rc::new(enc.finish()?));
		Ok(())
	}

	// Synchronize to a specific thread or board index
	fn synchronize(&mut self, dec: &mut Decoder) -> DynResult {
		let thread: u64 = dec.read_next()?;
		if thread != 0 && !bindings::thread_exists(thread)? {
			str_err!("invalid thread: {}", thread);
		}

		// Thread init data will be sent on the next pulse
		registry::set_client_thread(self.id, thread);

		Ok(())
	}

	// Decrease available solved captcha count, if available
	pub fn check_captcha(&mut self, solution: &[u8]) -> DynResult {
		if config::read(|c| c.captcha) {
			unimplemented!()
		}
		Ok(())
	}

	// Create a new thread and pass its ID to client
	fn create_thread(&mut self, dec: &mut Decoder) -> DynResult {
		let req: ThreadCreationReq = dec.read_next()?;
		check_len!(req.subject, 100);
		check_len!(req.tags, 3);
		for tag in req.tags.iter() {
			check_len!(tag, 20);
		}
		self.check_captcha(&req.captcha_solution)?;

		self.send(
			MessageType::CreateThreadAck,
			&bindings::insert_thread(
				req.subject,
				req.tags,
				self.key.as_ref().unwrap(),
			)?,
		)?;
		Ok(())
	}
}
