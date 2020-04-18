use super::common::DynResult;
use super::config;
use super::pulsar;
use super::{bindings, registry, str_err};
use protocol::{
	debug_log,
	payloads::{AuthKey, ThreadCreationReq},
	Decoder, Encoder, MessageType,
};
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

macro_rules! log_msg_in {
	($type:expr, $msg:expr) => {
		debug_log!(format!(">>> {:?}: {:?}", $type, $msg))
	};
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
		// Helper to make message handling through route!() more terse
		struct HandlerResult(DynResult);

		impl From<()> for HandlerResult {
			fn from(_: ()) -> HandlerResult {
				HandlerResult(Ok(()))
			}
		}

		impl From<DynResult> for HandlerResult {
			fn from(v: DynResult) -> HandlerResult {
				HandlerResult(v)
			}
		}

		impl Into<DynResult> for HandlerResult {
			fn into(self) -> DynResult {
				self.0
			}
		}

		// Separate function to enable type inference of payload type from
		// lambda argument type
		fn _route<'de, T, R>(
			dec: &'de mut Decoder,
			typ: MessageType,
			mut handler: impl FnMut(T) -> R,
		) -> DynResult
		where
			T: serde::Deserialize<'de> + std::fmt::Debug,
			R: Into<HandlerResult>,
		{
			let payload: T = dec.read_next()?;
			log_msg_in!(typ, payload);
			(handler(payload).into() as HandlerResult).into()
		}

		let mut dec = Decoder::new(buf)?;

		macro_rules! route {
			($type:expr, $($msg_type:ident => $handler:expr)+) => {
				match $type {
					$(
						MessageType::$msg_type => {
							_route(&mut dec, MessageType::$msg_type, $handler)?
						}
					)+
					_ => str_err!("unhandled message type: {:?}", $type),
				}
			};
		}

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
							let msg: protocol::payloads::Handshake =
								dec.read_next()?;
							log_msg_in!(MessageType::Handshake, msg);
							if msg.protocol_version != protocol::VERSION {
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
							let feed = dec.read_next()?;
							log_msg_in!(MessageType::Synchronize, feed);
							self.synchronize(feed)?;
							self.init_state = InitState::Synced;
						}
						InitState::Synced => route! { t,
							CreateThread => |req: ThreadCreationReq| {
								self.create_thread(req)
							}
							Synchronize => |feed: u64| {
								self.synchronize(feed)
							}
						},
					}
				}
			}
		}
	}

	// Send a private message to only this client
	fn send<T>(&self, t: MessageType, payload: &T) -> io::Result<()>
	where
		T: Serialize + std::fmt::Debug,
	{
		debug_log!(format!("<<< {:?}: {:?}", t, payload));

		let mut enc = Encoder::new(Vec::new());
		enc.write_message(t, payload)?;
		bindings::write_message(self.id, Arc::new(enc.finish()?));
		Ok(())
	}

	// Synchronize to a specific thread or board index
	fn synchronize(&mut self, feed: u64) -> DynResult {
		if feed != 0 && !bindings::thread_exists(feed)? {
			str_err!("invalid thread: {}", feed);
		}

		// Thread init data will be sent on the next pulse
		registry::set_client_thread(self.id, feed);

		// TODO: Lookup, if client has any open posts in thread and link them to
		// client

		self.send(MessageType::Synchronize, &feed)?;
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

	// Trim and replace String
	fn trim(src: &mut String) {
		let t = src.trim();
		// Don't always reallocate
		if src.len() != t.len() {
			*src = t.into();
		}
	}

	// Create a new thread and pass its ID to client
	fn create_thread(&mut self, mut req: ThreadCreationReq) -> DynResult {
		Self::trim(&mut req.subject);
		check_len!(req.subject, 100);
		check_len!(req.tags, 3);
		for mut tag in req.tags.iter_mut() {
			Self::trim(&mut tag);
			check_len!(tag, 20);
		}
		self.check_captcha(&req.captcha_solution)?;

		let id = bindings::insert_thread(&req.subject, &req.tags, &self.key)?;
		pulsar::create_thread(protocol::payloads::ThreadCreationNotice {
			id: id,
			subject: req.subject,
			tags: req.tags,
		})?;

		self.send(MessageType::CreateThreadAck, &id)?;
		Ok(())
	}
}
