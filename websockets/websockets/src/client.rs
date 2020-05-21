use crate::{bindings, common::DynResult, config, pulsar, registry, str_err};
use protocol::{
	debug_log,
	payloads::{
		post_body::TextPatch, Authorization, HandshakeReq, Signature,
		ThreadCreationReq,
	},
	Decoder, Encoder, MessageType,
};
use serde::Serialize;
use std::{net::IpAddr, sync::Arc};

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

struct OpenPost {
	id: u64,
	thread: u64,
	body: String,
	char_length: isize,
}

impl OpenPost {
	fn new(id: u64, thread: u64) -> Self {
		Self {
			id,
			thread,
			body: String::new(),
			char_length: 0,
		}
	}
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

	// Post the client is currently editing
	open_post: Option<OpenPost>,
}

// Return with invalid length error
macro_rules! err_invalid_length {
	($val:expr, $len:expr) => {
		str_err!("invalid {} length: {}", stringify!($val), $len);
	};
}

// Assert collection length
#[rustfmt::skip]
macro_rules! check_len {
	// Assert collection length greater than 1 and smaller than $max
	($val:expr, $max:expr) => {
		check_len!($val, 1, $max)
	};
	// Assert collection length greater than $min and smaller than $max
	($val:expr, $min:expr, $max:expr) => {{
		let l = $val.len();
		if l < $min || l > $max {
			err_invalid_length!($val, l)
		}
	}};
}

// Assert unicode string character length. Returns the length.
#[rustfmt::skip]
macro_rules! check_unicode_len {
	// Assert string length greater than 1 and smaller than $max
	($val:expr, $max:expr) => {
		check_unicode_len!($val, 1, $max)
	};
	// Assert string length greater than $min and smaller than $max
	($val:expr, $min:expr, $max:expr) => {{
		let l = $val.chars().count();
		if l < $min || l > $max {
			err_invalid_length!($val, l)
		}
		l
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
			open_post: Default::default(),
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
					use ConnState::*;
					use MessageType::*;

					#[rustfmt::skip]
					macro_rules! expect {
						($type:tt) => {
							if t != $type {
								str_err!(concat!(
									"expected ",
									stringify!($type)
								));
							}
						};
					}

					first = false;
					match &self.conn_state {
						Connected => {
							expect!(Handshake);
							self.handle_handshake(&mut dec)?;
						}
						RequestedReshake { desc, pub_key } => {
							expect!(Handshake);
							let desc = desc.clone();
							let pub_key = pub_key.clone();
							self.handle_reshake(&mut dec, desc, pub_key)?;
						}
						AcceptedHandshake(desc) => {
							expect!(Synchronize);
							let feed = dec.read_next()?;
							log_msg_in!(MessageType::Synchronize, feed);
							self.conn_state = Synchronizing(desc.clone());
							self.synchronize(feed)?;
						}
						Synchronizing(_) => route! { t,
							CreateThread => |req: ThreadCreationReq| {
								self.create_thread(req)
							}
							Synchronize => |feed: u64| {
								self.synchronize(feed)
							}
							Append => |s: String| {
								let n = check_unicode_len!(s, 2000);
								self.update_body(n as isize, n, |b| {
									*b += &s;
									Ok(())
								})
							}
							Backspace => |n: u16| {
								self.backspace(n as usize)
							}
							PatchPostBody => |req: TextPatch| {
								self.patch_body(req)
							}
						},
					}
				}
			}
		}
	}

	// Send a private message to only this client
	fn send<T>(&self, t: MessageType, payload: &T) -> std::io::Result<()>
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
			// TODO: Use pub key for spam detection bans
			todo!()
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
		check_unicode_len!(req.subject, 100);
		check_len!(req.tags, 3);
		for mut tag in req.tags.iter_mut() {
			Self::trim(&mut tag);
			check_unicode_len!(tag, 20);
		}
		self.check_captcha(&req.captcha_solution)?;

		let id = bindings::insert_thread(
			&req.subject,
			&req.tags,
			self.public_key_id()?,
			Self::empty_body_json(),
		)?;

		// Ensures old post non-existence records do not persist indefinitely.
		crate::body::cache_location(id, id, 0);

		pulsar::create_thread(protocol::payloads::ThreadCreationNotice {
			id: id,
			subject: req.subject,
			tags: req.tags,
		})?;

		self.send(MessageType::CreateThreadAck, &id)?;
		self.open_post = OpenPost::new(id, id).into();
		Ok(())
	}

	// Reduce open post text body size by n chars from the back
	fn backspace(&mut self, n: usize) -> DynResult {
		if n == 0 {
			str_err!("backspace size must be at least 1")
		}
		self.update_body(-(n as isize), n, |s| {
			let mut removed = 0;
			for (i, b) in s.as_bytes().iter().enumerate().rev() {
				if Self::is_char_start(*b) {
					removed += 1;
					if removed == n {
						s.truncate(i);
						return Ok(());
					}
				}
			}
			Ok(())
		})
	}

	// Reports whether the byte could be the first byte of an encoded,
	// possibly invalid character. Second and subsequent bytes always have the
	// top two bits set to 10.
	fn is_char_start(b: u8) -> bool {
		b & 0xC0 != 0x80
	}

	// Apply diff to text body
	fn patch_body(&mut self, req: TextPatch) -> DynResult {
		let insert_len = check_unicode_len!(req.insert, 2000);
		if insert_len == 0 && req.remove == 0 {
			str_err!("patch is a NOP")
		}
		self.update_body(
			insert_len as isize - req.remove as isize,
			insert_len + req.remove as usize,
			|s| {
				// Get the byte position of the requested character position
				fn byte_pos(s: &str, needed_char_pos: usize) -> Option<usize> {
					let mut char_pos: isize = -1;
					for (i, b) in s.as_bytes().iter().enumerate() {
						if Client::is_char_start(*b) {
							char_pos += 1;
							if char_pos as usize == needed_char_pos {
								return Some(i);
							}
						}
					}
					None
				}

				let end = s.split_off(
					byte_pos(&s, req.position as usize)
						.ok_or("patch position out of bounds")?,
				);
				*s += &req.insert;
				*s += &end[..byte_pos(&end, req.remove as usize)
					.ok_or("char count to remove out of bounds")?];
				Ok(())
			},
		)
	}

	// Update post body, sync to various services and DB and performs error
	// handling
	//
	// len_diff: how much in Unicode chars would the length of the body change
	// affected: number of Unicode characters affected by the mutation
	// modify: modifies text body
	fn update_body(
		&mut self,
		len_diff: isize,
		affected: usize,
		modify: impl Fn(&mut String) -> Result<(), &'static str>,
	) -> DynResult {
		let p = match &mut self.open_post {
			Some(p) => p,
			None => return Err("no post open".into()),
		};

		p.char_length += len_diff;
		if p.char_length < 0 || p.char_length > 2000 {
			str_err!("body length would exceed bounds")
		}

		modify(&mut p.body)?;

		pulsar::set_open_body(p.id, p.thread, p.body.clone())?;
		bindings::increment_spam_score(
			self.public_key_id()?,
			affected * crate::config::read(|c| c.spam_scores.character),
		);

		Ok(())
	}

	// Cached empty body JSON representation
	//
	// Non-useless const fn when?
	fn empty_body_json() -> &'static [u8] {
		use std::sync::Once;

		static ONCE: Once = Once::new();
		static mut BODY: Option<Vec<u8>> = None;
		ONCE.call_once(|| {
			unsafe {
				BODY = Some(
					serde_json::to_vec(
						&protocol::payloads::post_body::Node::Empty,
					)
					.expect("failed to generate empty body JSON"),
				)
			};
		});

		unsafe { BODY.as_ref().unwrap() }
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
