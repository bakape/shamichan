use super::{client::Client, str_err};
use crate::{
	body::{cache_locations, KnownPostLocation},
	config, db,
	feeds::{self, AnyFeed, PostLocation, ThreadFeed},
	message::Message,
	mt_context::{AsyncHandler, MTAddr, MTContext},
	registry,
	util::{self, DynResult},
};
use actix::{Actor, Addr};
use actix_web::web::Bytes;
use async_trait::async_trait;
use common::{
	payloads::{
		self, post_body::TextPatch, Authorization, HandshakeReq,
		PostCreationReq, Signature, ThreadCreationReq,
	},
	Decoder, Encoder, MessageType,
};
use serde::Serialize;
use std::sync::Arc;

/// Return with invalid length error
macro_rules! err_invalid_length {
	($val:expr, $len:expr) => {
		str_err!("invalid {} length: {}", stringify!($val), $len);
	};
}

/// Assert collection length
///
/// $val: expression to check length of
/// $min: minimum length; defaults to 1
/// $max: maximum length
#[rustfmt::skip]
macro_rules! check_len {
	($val:expr, $max:expr) => {
		check_len!($val, 1, $max)
	};
	($val:expr, $min:expr, $max:expr) => {{
		let l = $val.len();
		if l < $min || l > $max {
			err_invalid_length!($val, l)
		}
	}};
}

/// Assert unicode string character length. Returns the length.
///
/// $val: expression to check length of
/// $min: minimum length; defaults to 1
/// $max: maximum length
#[rustfmt::skip]
macro_rules! check_unicode_len {
	($val:expr, $max:expr) => {
		check_unicode_len!($val, 1, $max)
	};
	($val:expr, $min:expr, $max:expr) => {{
		let l = $val.chars().count();
		if l < $min || l > $max {
			err_invalid_length!($val, l)
		}
		l
	}};
}

/// Public key public and private ID set
#[derive(Clone, Default, Debug)]
struct PubKeyDesc {
	/// Public key private ID used to sign messages by the client
	priv_id: u64,

	/// Public key public ID used to sign messages by the client
	pub_id: uuid::Uuid,
}

#[derive(Debug)]
struct OpenPost {
	thread: u64,
	loc: feeds::PostLocation,
	body: Vec<char>,
	feed: MTAddr<ThreadFeed>,
}

/// Client connection state
#[derive(Debug)]
enum ConnState {
	/// Freshly established a WS connection
	Connected,

	/// Sent handshake message and it was accepted
	AcceptedHandshake,

	/// Public key already registered. Requested client to send a HandshakeReq
	/// with Authorization::Saved.
	RequestedReshake { pub_key: Vec<u8> },

	/// Client synchronized to a feed
	Synchronized { id: u64, feed: AnyFeed },
}

/// Handles incoming messages asynchronously\
#[derive(Debug)]
pub struct MessageHandler {
	/// Immutable client state set on client creation
	state: Arc<super::State>,

	/// Client connection state
	conn_state: ConnState,

	/// Post the client is currently editing
	open_post: Option<OpenPost>,

	/// Public key public and private ID set
	pub_key: PubKeyDesc,

	/// Calling Client address
	client: Addr<Client>,

	/// Message being written to
	message: Option<Encoder>,
}

impl Actor for MessageHandler {
	type Context = MTContext<Self>;
}

/// Handle received message and send result back to parent client
pub struct HandleMessage(pub Bytes);

/// Result of asynchronously processing a message
#[derive(actix::Message)]
#[rtype(result = "()")]
pub struct MessageResult(pub DynResult<Option<Message>>);

#[async_trait]
impl AsyncHandler<HandleMessage> for MessageHandler {
	type Error = util::Err;

	async fn handle(
		&mut self,
		HandleMessage(buf): HandleMessage,
		_: &mut <Self as Actor>::Context,
	) -> Result<(), Self::Error> {
		let msg = match self.handle_messages(buf).await {
			Ok(_) => Ok(match self.message.take() {
				Some(m) => Some(m.finish()?.into()),
				None => {
					// No need to respond, if no message or error to send.
					// Avoids some message handling and synchronization costs.
					return Ok(());
				}
			}),
			Err(e) => Err(e),
		};
		self.client.do_send(MessageResult(msg));
		Ok(())
	}
}

impl MessageHandler {
	/// Construct new handler for handling one specific message
	pub(super) fn new(state: Arc<super::State>, client: Addr<Client>) -> Self {
		Self {
			state,
			client,
			conn_state: ConnState::Connected,
			open_post: None,
			message: None,
			pub_key: Default::default(),
		}
	}

	/// Decode a message and return the decoded type
	#[inline]
	fn decode<T>(t: MessageType, dec: &mut Decoder) -> DynResult<T>
	where
		T: for<'de> serde::Deserialize<'de> + std::fmt::Debug,
	{
		let payload: T = dec.read_next()?;
		common::log_msg_in!(t, payload);
		Ok(payload)
	}

	/// Handle received messages
	async fn handle_messages(&mut self, buf: Bytes) -> DynResult {
		let mut dec = Decoder::new(&buf)?;
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
								str_err!(
									"expected message type {:?}, got {:?}",
									$type,
									t,
								);
							}
						};
					}

					first = false;
					match &self.conn_state {
						Connected => {
							expect!(Handshake);
							self.handle_handshake(&mut dec).await?;
							self.send(CurrentTime, &util::now())?;
							self.send(Configs, &config::get().public)?;
						}
						RequestedReshake { pub_key } => {
							expect!(Handshake);
							let pk = pub_key.clone();
							self.handle_reshake(&mut dec, &pk)?;
						}
						AcceptedHandshake | Synchronized { .. } => {
							self.handle_message_after_handshake(t, &mut dec)
								.await?;
						}
					}
				}
			}
		}
	}

	/// Handle a received message after a successful handshake
	async fn handle_message_after_handshake(
		&mut self,
		t: MessageType,
		dec: &mut Decoder,
	) -> DynResult {
		use MessageType::*;

		macro_rules! decode {
			() => {
				Self::decode(t, dec)?
			};
		}

		macro_rules! skip_payload {
			() => {
				dec.skip_next();
				common::log_msg_in!(t, ());
			};
		}

		match t {
			InsertThread => self.insert_thread(decode!()).await,
			Synchronize => self.synchronize(decode!()).await,
			InsertPost => self.insert_post(decode!()).await,
			Append => {
				let ch = decode!();
				self.update_body(1, |b| {
					b.push(ch);
					Ok(())
				})
			}
			Backspace => {
				skip_payload!();
				self.update_body(1, |b| {
					b.pop();
					Ok(())
				})
			}
			PatchPostBody => self.patch_body(decode!()),
			Page => self.fetch_page(decode!()),
			UsedTags => {
				skip_payload!();
				self.state
					.index_feed
					.do_send(feeds::UsedTags(self.client.clone()));
				Ok(())
			}
			ClosePost => {
				skip_payload!();
				self.close_post().await
			}
			_ => str_err!("unhandled message type: {:?}", t),
		}
	}

	/// Buffer a private message to be sent only to this client
	fn send<T>(&mut self, t: MessageType, payload: &T) -> std::io::Result<()>
	where
		T: Serialize + std::fmt::Debug,
	{
		log::debug!("<<< {:?}: {:?}", t, payload);
		self.message
			.get_or_insert_with(|| Default::default())
			.write_message(t, payload)
	}

	/// Synchronize to a specific thread or board index
	#[cold]
	async fn synchronize(&mut self, feed: u64) -> DynResult {
		self.conn_state = ConnState::Synchronized {
			id: feed,
			feed: self
				.state
				.registry
				.send(registry::SetFeed {
					client: self.state.id,
					feed,
				})
				.await??,
		};

		// TODO: attempt to reclaim an open post lost to disconnection, if any
		//  specified by client

		Ok(())
	}

	/// Fetch a page from a currently synced to feed
	fn fetch_page(&mut self, page: i32) -> DynResult {
		match &self.conn_state {
			ConnState::Synchronized { feed, .. } => match feed {
				AnyFeed::Index(_) => {
					str_err!("can not fetch pages on index feed")
				}
				AnyFeed::Thread(f) => {
					f.do_send(feeds::FetchPage {
						id: page,
						client: self.client.clone(),
					});
					Ok(())
				}
			},
			_ => {
				str_err!("need to be synchronized to a thread to request pages")
			}
		}
	}

	/// Validates a solved captcha
	pub fn check_captcha(&mut self, solution: &[u8]) -> DynResult {
		if config::get().public.enable_antispam {
			check_len!(solution, 4);

			// TODO: Use pub key for spam detection bans
			// TODO: validate solution
		}
		Ok(())
	}

	/// Trim and replace String
	#[inline]
	fn trim(src: &mut String) {
		let t = src.trim();
		// Don't always reallocate
		if src.len() != t.len() {
			*src = t.into();
		}
	}

	/// Assert client does not already have an open post
	#[inline]
	fn assert_no_open_post(&self) -> Result<(), String> {
		if self.open_post.is_some() {
			str_err!("already have an open post")
		}
		Ok(())
	}

	/// Create a new thread and pass its ID to client
	async fn insert_thread(&mut self, mut req: ThreadCreationReq) -> DynResult {
		// TODO: Lock new thread form, if postform is open
		self.assert_no_open_post()?;

		Self::trim(&mut req.subject);
		check_unicode_len!(req.subject, 100);

		check_len!(req.tags, 3);
		for tag in req.tags.iter_mut() {
			Self::trim(tag);
			*tag = tag.to_lowercase();
			check_unicode_len!(tag, 20);
		}
		if req
			.tags
			.iter()
			.collect::<std::collections::BTreeSet<_>>()
			.len() != req.tags.len()
		{
			str_err!("tag set contains duplicates")
		}
		req.tags.sort();

		let [name, trip] = Self::parse_name(req.opts.name)?;
		self.check_captcha(&req.captcha_solution)?;
		let id = db::insert_thread(&mut db::ThreadInsertParams {
			subject: &req.subject,
			tags: &mut req.tags,
			op: db::PostInsertParams {
				public_key: self.pub_key.priv_id.into(),
				name: name.as_ref().map(AsRef::as_ref),
				trip: trip.as_ref().map(AsRef::as_ref),
				flag: None, // TODO
				body: &common::payloads::post_body::Node::Empty,
			},
		})
		.await?;

		// Ensures old post non-existence records do not persist indefinitely
		cache_locations(std::iter::once(KnownPostLocation {
			id,
			thread: id,
			page: 0,
		}));

		let feed = self
			.state
			.registry
			.send(registry::InsertThread(feeds::InsertThread {
				id,
				subject: req.subject,
				tags: req.tags,
				opts: payloads::PostCreationOpts {
					name,
					trip,
					flag: None, // TODO
				},
			}))
			.await?;
		self.send(MessageType::InsertThreadAck, &id)?;
		self.open_post = Some(OpenPost {
			loc: feeds::PostLocation { id, page: 0 },
			thread: id,
			body: Default::default(),
			feed,
		});

		Ok(())
	}

	/// Create a new post in a thread and pass its ID to client
	async fn insert_post(&mut self, req: PostCreationReq) -> DynResult {
		self.assert_no_open_post()?;

		// TODO: captcha checks
		// if bindings::need_captcha(self.pub_key.priv_id)? {
		// 	self.send(MessageType::NeedCaptcha, &())?;
		// 	return Ok(());
		// }

		let [name, trip] = Self::parse_name(req.opts.name)?;
		let (id, page) = db::insert_post(
			req.thread,
			req.sage,
			&db::PostInsertParams {
				public_key: self.pub_key.priv_id.into(),
				name: name.as_ref().map(AsRef::as_ref),
				trip: trip.as_ref().map(AsRef::as_ref),
				flag: None, // TODO
				body: &common::payloads::post_body::Node::Empty,
			},
		)
		.await?;

		// Ensures old post non-existence records do not persist indefinitely
		cache_locations(std::iter::once(KnownPostLocation {
			id,
			thread: req.thread,
			page,
		}));

		// Don't fetch feed address, if open post in same feed as synced
		let feed = match &self.conn_state {
			ConnState::Synchronized {
				id,
				feed: feeds::AnyFeed::Thread(f),
			} if id == &req.thread => f.clone(),
			_ => {
				self.state
					.registry
					.send(registry::GetFeed(req.thread))
					.await??
			}
		};

		// At least try to make it arrive before the post insert
		self.send(MessageType::InsertPostAck, &id)?;

		feed.do_send(feeds::InsertPost {
			id,
			thread: req.thread,
			page,
			opts: payloads::ReplyCreationOpts {
				sage: req.sage,
				post_opts: payloads::PostCreationOpts {
					name,
					trip,
					flag: None, // TODO
				},
			},
		});

		self.open_post = Some(OpenPost {
			loc: feeds::PostLocation { id, page },
			thread: req.thread,
			body: Default::default(),
			feed,
		});

		Ok(())
	}

	/// Apply diff to text body
	fn patch_body(&mut self, req: TextPatch) -> DynResult {
		if req.insert.len() > 2000 {
			str_err!("patch too long")
		}
		if req.insert.len() == 0 && req.remove == 0 {
			str_err!("patch is a NOP")
		}

		let affected = req.insert.len() + req.remove as usize;
		self.update_body(affected, move |b| {
			if req.position as usize > b.len() {
				return Err(format!(
					"splice position {} exceeds body length {}",
					req.position,
					b.len()
				));
			}

			let mut new = Vec::with_capacity(req.estimate_new_size(b.len()));
			req.apply(&mut new, b.iter().copied());
			*b = new;

			if b.len() > 2000 {
				str_err!("body length exceeds bounds")
			}

			Ok(())
		})
	}

	/// Update post body, sync to various services and DB and performs error
	/// handling
	//
	/// affected: number of Unicode characters affected by the mutation
	/// modify: modifies text body
	fn update_body(
		&mut self,
		affected: usize,
		modify: impl Fn(&mut Vec<char>) -> Result<(), String>,
	) -> DynResult {
		match &mut self.open_post {
			Some(p) => {
				modify(&mut p.body)?;
				if p.body.len() > 2000 {
					str_err!("body length exceeds bounds")
				}

				p.feed.do_send(feeds::SetBody {
					loc: p.loc.clone(),
					body: p.body.clone(),
				});
				// TODO: port spam scores to Rust
				// bindings::increment_spam_score(
				// 	self.pub_key.priv_id,
				// 	affected * crate::config::read(|c| c.spam_scores.character),
				// );

				Ok(())
			}
			None => Err("no post open".into()),
		}
	}

	fn decode_handshake(dec: &mut Decoder) -> DynResult<HandshakeReq> {
		let req: HandshakeReq = dec.read_next()?;
		common::log_msg_in!(MessageType::Handshake, req);
		if req.protocol_version != common::VERSION {
			str_err!("protocol version mismatch: {}", req.protocol_version);
		}
		Ok(req)
	}

	async fn handle_handshake(&mut self, dec: &mut Decoder) -> DynResult {
		use common::payloads::{HandshakeRes, PubKeyStatus};

		match Self::decode_handshake(dec)?.auth {
			Authorization::NewPubKey(pub_key) => {
				check_len!(pub_key, 1 << 10);
				let (priv_id, pub_id, fresh) =
					db::register_public_key(&pub_key).await?;

				self.pub_key = PubKeyDesc {
					priv_id,
					pub_id: pub_id.clone(),
				};
				if fresh {
					self.state
						.registry
						.send(registry::SetPublicKey {
							client: self.state.id,
							pub_key: priv_id,
						})
						.await??;
					self.conn_state = ConnState::AcceptedHandshake;
				} else {
					self.conn_state = ConnState::RequestedReshake { pub_key };
				}

				self.send(
					MessageType::Handshake,
					&HandshakeRes {
						id: pub_id,
						status: if fresh {
							PubKeyStatus::Accepted
						} else {
							PubKeyStatus::NeedResend
						},
					},
				)?;
			}
			Authorization::Saved {
				id: pub_id,
				nonce,
				signature,
			} => {
				match db::get_public_key(&pub_id).await? {
					Some((priv_id, pub_key)) => {
						self.pub_key = PubKeyDesc { priv_id, pub_id };
						self.handle_auth_saved(
							nonce,
							signature,
							pub_key.as_ref(),
						)?;
					}
					None => {
						self.send(
							MessageType::Handshake,
							&HandshakeRes {
								id: pub_id,
								status: PubKeyStatus::NotFound,
							},
						)?;
					}
				};
			}
		}
		Ok(())
	}

	/// Handle Authorization::Saved in handshake request
	fn handle_auth_saved(
		&mut self,
		nonce: [u8; 32],
		signature: Signature,
		pub_key: &[u8],
	) -> DynResult {
		use common::payloads::{HandshakeRes, PubKeyStatus};

		let pk = openssl::pkey::PKey::from_rsa(
			openssl::rsa::Rsa::public_key_from_der(pub_key)?,
		)?;
		let mut v = openssl::sign::Verifier::new(
			openssl::hash::MessageDigest::sha256(),
			&pk,
		)?;
		v.update(self.pub_key.pub_id.as_bytes())?;
		v.update(&nonce)?;
		if !v.verify(&signature.0)? {
			str_err!("invalid signature");
		}

		self.send(
			MessageType::Handshake,
			&HandshakeRes {
				id: self.pub_key.pub_id,
				status: PubKeyStatus::Accepted,
			},
		)?;
		self.conn_state = ConnState::AcceptedHandshake;
		Ok(())
	}

	/// Handle repeated handshake after request by server
	fn handle_reshake(
		&mut self,
		mut dec: &mut Decoder,
		pub_key: &[u8],
	) -> DynResult {
		match Self::decode_handshake(&mut dec)?.auth {
			Authorization::Saved {
				id: pub_id,
				nonce,
				signature,
			} => {
				if pub_id != self.pub_key.pub_id {
					str_err!("different public key public id in reshake");
				}
				self.handle_auth_saved(nonce, signature, pub_key)?;
			}
			_ => str_err!("invalid authorization variant"),
		}
		Ok(())
	}

	/// Parse post name field in to name and tripcode
	fn parse_name(
		mut src: String,
	) -> Result<[Option<String>; 2], &'static str> {
		use tripcode::{FourchanNonescaping, TripcodeGenerator};

		Ok(match src.len() {
			0 => Default::default(),
			l if l > 50 => Err("name too long")?,
			_ => {
				Self::trim(&mut src);
				match src.as_bytes().iter().position(|b| b == &b'#') {
					Some(i) if i != src.len() - 1 => {
						let trip = FourchanNonescaping::generate(&src[i + 1..]);
						src.truncate(i);
						[Some(src), Some(trip)]
					}
					_ => [Some(src), None],
				}
			}
		})
	}

	/// Close the currently open post
	async fn close_post(&mut self) -> DynResult {
		use common::payloads::post_body::{Command, Node, PendingNode};

		let p = self.open_post.take().ok_or_else(|| "no post open")?;
		let mut body =
			crate::body::parse(&p.body.iter().collect::<String>(), false);

		#[async_recursion::async_recursion]
		async fn finalize_pending(n: &mut Node) -> DynResult {
			use rand::prelude::*;

			match n {
				Node::Children(ch) => {
					for ch in ch.iter_mut() {
						finalize_pending(ch).await?
					}
				}
				Node::Pending(p) => {
					*n = match p {
						PendingNode::Autobahn(h) => {
							Node::Command(Command::Autobahn(*h))
						}
						PendingNode::Countdown(s) => {
							Node::Command(Command::Countdown {
								start: util::now(),
								secs: *s,
							})
						}
						PendingNode::Dice {
							offset,
							faces,
							rolls,
						} => Node::Command(Command::Dice {
							offset: *offset,
							faces: *faces,
							results: {
								let mut res =
									Vec::with_capacity(*rolls as usize);

								let mut rng = thread_rng();
								for _ in 0..*rolls {
									res.push(rng.gen::<u16>() % *faces + 1);
								}

								res
							},
						}),
						PendingNode::EightBall => {
							// TODO: read eightball choices from thread configs
							static CHOICES: [&str; 5] = [
								"Yes",
								"No",
								"Maybe",
								"Anta baka?",
								"Hell yeah, motherfucker!",
							];

							Node::Command(Command::EightBall(
								CHOICES[random::<usize>() % CHOICES.len()]
									.into(),
							))
						}
						PendingNode::Flip => {
							Node::Command(Command::Flip(random()))
						}
						PendingNode::PostLink(id) => {
							match crate::body::post_location(*id).await? {
								Some((thread, page)) => Node::PostLink {
									id: *id,
									thread,
									page,
								},
								None => Node::Text(format!(">>{}", id)),
							}
						}
						PendingNode::Pyu => Node::Command(Command::Pyu(
							crate::db::increment_pcount().await?,
						)),
						PendingNode::PCount => Node::Command(Command::PCount(
							crate::db::get_pcount().await?,
						)),
					}
				}
				_ => (),
			};
			Ok(())
		}

		finalize_pending(&mut body).await?;

		crate::db::close_post(p.loc.id, &body).await?;
		p.feed.do_send(crate::feeds::ClosePost {
			loc: PostLocation {
				page: p.loc.page,
				id: p.loc.id,
			},
			body,
		});

		Ok(())
	}
}
