use super::common::DynResult;
use super::config;
use super::pulsar;
use super::{bindings, registry, str_err};
use protocol::{
	debug_log,
	payloads::{Authorization, HandshakeReq, Signature, ThreadCreationReq},
	Decoder, Encoder, MessageType,
};
use serde::Serialize;
use std::io;
use std::net::IpAddr;
use std::sync::Arc;

// Public key public and private ID set
#[derive(Clone)]
struct PubKeyDesc {
	// Public key private ID used to sign messages by the client
	priv_id: u64,

	// Public key public ID used to sign messages by the client
	pub_id: uuid::Uuid,
}

// Client connection state
enum ConnState {
	// Freshly established a WS connection
	Connected,

	// Sent handshake message and it was accepted
	AcceptedHandshake(PubKeyDesc),

	// Public key already registered. Requested client to send a HandshakeReq
	// with Authorization::Saved.
	RequestedReshake { desc: PubKeyDesc, pub_key: Vec<u8> },

	// Client synchronizing to a feed
	Synchronizing(PubKeyDesc),
}

// Maps to a websocket client on the Go side
pub struct Client {
	// ID of client used in various registries
	id: u64,

	// IP of client connection
	//
	// TODO: Use this for bans
	ip: IpAddr,

	// Client connection state
	conn_state: ConnState,
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
			conn_state: ConnState::Connected,
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
					#[rustfmt::skip]
					macro_rules! expect {
						($type:ident) => {
							if t != MessageType::$type {
								str_err!(concat!(
									"expected ",
									stringify!(MessageType::$type)
								));
							}
						};
					}

					first = false;
					match &self.conn_state {
						ConnState::Connected => {
							expect!(Handshake);
							self.handle_handshake(&mut dec)?;
						}
						ConnState::RequestedReshake { desc, pub_key } => {
							expect!(Handshake);
							let desc = desc.clone();
							let pub_key = pub_key.clone();
							self.handle_reshake(&mut dec, desc, pub_key)?;
						}
						ConnState::AcceptedHandshake(desc) => {
							expect!(Synchronize);
							let feed = dec.read_next()?;
							log_msg_in!(MessageType::Synchronize, feed);
							self.conn_state =
								ConnState::Synchronizing(desc.clone());
							self.synchronize(feed)?;
						}
						ConnState::Synchronizing(_) => route! { t,
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

		let id = bindings::insert_thread(
			&req.subject,
			&req.tags,
			self.public_key_id()?,
		)?;
		pulsar::create_thread(protocol::payloads::ThreadCreationNotice {
			id: id,
			subject: req.subject,
			tags: req.tags,
		})?;

		self.send(MessageType::CreateThreadAck, &id)?;
		Ok(())
	}

	// Get public key private ID, if client is synchronized
	fn public_key_id(&self) -> Result<u64, &'static str> {
		match &self.conn_state {
			ConnState::Synchronizing(desc) => Ok(desc.priv_id),
			_ => Err("client not synchronizing"),
		}
	}

	fn decode_handshake(dec: &mut Decoder) -> DynResult<HandshakeReq> {
		let req: HandshakeReq = dec.read_next()?;
		log_msg_in!(MessageType::Handshake, req);
		if req.protocol_version != protocol::VERSION {
			str_err!("protocol version mismatch: {}", req.protocol_version);
		}
		Ok(req)
	}

	fn handle_handshake(&mut self, dec: &mut Decoder) -> DynResult {
		match Self::decode_handshake(dec)?.auth {
			Authorization::NewPubKey(pub_key) => {
				check_len!(pub_key, 1 << 10);
				let (priv_id, pub_id, fresh) =
					bindings::register_public_key(&pub_key)?;

				let desc = PubKeyDesc {
					priv_id,
					pub_id: pub_id.clone(),
				};
				if fresh {
					registry::set_client_key(self.id, priv_id);
					self.conn_state = ConnState::AcceptedHandshake(desc);
				} else {
					self.conn_state =
						ConnState::RequestedReshake { pub_key, desc };
				}

				self.send(
					MessageType::Handshake,
					&protocol::payloads::HandshakeRes {
						need_resend: !fresh,
						id: pub_id,
					},
				)?;
			}
			Authorization::Saved {
				id: pub_id,
				nonce,
				signature,
			} => {
				let (priv_id, pub_key) = bindings::get_public_key(pub_id)?;
				self.handle_auth_saved(
					PubKeyDesc { priv_id, pub_id },
					nonce,
					signature,
					pub_key.as_ref(),
				)?;
			}
		}
		Ok(())
	}

	// Handle Authorization::Saved in handshake request
	fn handle_auth_saved(
		&mut self,
		desc: PubKeyDesc,
		nonce: [u8; 32],
		signature: Signature,
		pub_key: &[u8],
	) -> DynResult {
		let pk = openssl::pkey::PKey::from_rsa(
			openssl::rsa::Rsa::public_key_from_der(pub_key)?,
		)?;
		let mut v = openssl::sign::Verifier::new(
			openssl::hash::MessageDigest::sha256(),
			&pk,
		)?;
		v.update(desc.pub_id.as_bytes())?;
		v.update(&nonce)?;
		if !v.verify(&signature.0)? {
			str_err!("invalid signature");
		}

		self.send(
			MessageType::Handshake,
			&protocol::payloads::HandshakeRes {
				need_resend: false,
				id: desc.pub_id,
			},
		)?;
		self.conn_state = ConnState::AcceptedHandshake(desc);
		Ok(())
	}

	// Handle repeated handshake after request by server
	fn handle_reshake(
		&mut self,
		mut dec: &mut Decoder,
		desc: PubKeyDesc,
		pub_key: Vec<u8>,
	) -> DynResult {
		match Self::decode_handshake(&mut dec)?.auth {
			Authorization::Saved {
				id: pub_id,
				nonce,
				signature,
			} => {
				if pub_id != desc.pub_id {
					str_err!("different public key public id in reshake");
				}
				self.handle_auth_saved(desc, nonce, signature, &pub_key)?;
			}
			_ => str_err!("invalid authorization variant"),
		}
		Ok(())
	}
}
