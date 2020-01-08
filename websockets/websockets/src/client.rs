use super::common::DynResult;
use super::config;
use super::{bindings, registry, str_err};
use protocol::AuthKey;
use protocol::*;
use serde::Serialize;
use std::io;
use std::net::IpAddr;
use std::sync::Arc;

// Client initialization state
enum InitState {
	Connected,
	SentHandshake,
	Synced,
}

// Maps to a websocket client on the Go side
pub struct Client {
	// ID of client used in various registries
	id: u64,

	// IP of client connection
	//
	// TODO: Use this for bans
	ip: IpAddr,

	// Client initialization state
	init_state: InitState,

	// Used to authenticate the client
	key: AuthKey,
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
			init_state: InitState::Connected,
			key: Default::default(),
		}
	}

	// Handle received message
	pub fn receive_message(&mut self, buf: &[u8]) -> DynResult {
		let mut dec = Decoder::new(buf)?;
		let mut first = true;
		loop {
			match dec.peek_type() {
				None => {
					if first {
						str_err!("empty message received");
					}
					return Ok(());
				}
				Some(t) => {
					first = false;
					match self.init_state {
						InitState::Connected => {
							if t != MessageType::Handshake {
								str_err!("first message must be handshake");
							}
							let msg: Handshake = dec.read_next()?;
							debug_log!("received handshake", msg);
							if msg.protocol_version != VERSION {
								str_err!(
									"protocol version mismatch: {}",
									msg.protocol_version
								);
							}
							registry::set_client_key(self.id, msg.key.clone());
							self.key = msg.key;
							self.init_state = InitState::SentHandshake;
						}
						InitState::SentHandshake => {
							if t != MessageType::Synchronize {
								str_err!("second message must be sync request");
							}
							self.synchronize(&mut dec)?;
							self.init_state = InitState::Synced;
						}
						InitState::Synced => match t {
							MessageType::CreateThread => {
								self.create_thread(&mut dec)?
							}
							_ => str_err!("unhandled message type: {:?}", t),
						},
					}
				}
			}
		}
	}

	// Send a private message to only this client
	fn send(&self, t: MessageType, payload: &impl Serialize) -> io::Result<()> {
		let mut enc = Encoder::new(Vec::new());
		enc.write_message(t, payload)?;
		bindings::write_message(self.id, Arc::new(enc.finish()?));
		Ok(())
	}

	// Synchronize to a specific thread or board index
	fn synchronize(&mut self, dec: &mut Decoder) -> DynResult {
		let thread: u64 = dec.read_next()?;
		debug_log!("received sync req", thread);
		if thread != 0 && !bindings::thread_exists(thread)? {
			str_err!("invalid thread: {}", thread);
		}

		// Thread init data will be sent on the next pulse
		registry::set_client_thread(self.id, thread);

		// TODO: Send open post and moderation data
		// TODO: Lookup, if client has any open posts in thread and send their
		// dat, if any

		self.send(MessageType::Synchronize, &thread)?;
		Ok(())
	}

	// Decrease available solved captcha count, if available
	pub fn check_captcha(&mut self, _solution: &[u8]) -> DynResult {
		if config::read(|c| c.captcha) {
			// TODO: Use IP for spam detection bans
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
			&bindings::insert_thread(req.subject, req.tags, &self.key)?,
		)?;
		Ok(())
	}
}
