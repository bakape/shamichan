use super::{
	state::{self, KeyPair},
	util,
};
use common::{debug_log, Decoder, Encoder, MessageType};
use serde::Serialize;
use std::{collections::HashSet, fmt::Debug};
use yew::{
	agent::{Agent, AgentLink, Context, Dispatched, HandlerId},
	html,
	services::{
		console::ConsoleService,
		timeout::{TimeoutService, TimeoutTask},
	},
	Bridge, Bridged, Component, ComponentLink, Html,
};

// TODO: break up into submodules

// TODO: after a disconnect, the first message must be a sync message

// TODO: send open post reclamation request with a full text body for any open
// post on reconnect (via notification). Server-side it should be handled as
// Client -> ThreadFeed -> Client -> websocket response (confirmation or
// failure)

fn encode_msg<T>(
	enc: &mut Encoder,
	t: MessageType,
	payload: &T,
) -> std::io::Result<()>
where
	T: Serialize + Debug,
{
	debug_log!(format!("<<< {:?}: {:?}", t, payload));
	enc.write_message(t, payload)
}

/// Send a message over websocket.
/// Log any encoding errors (there should not be any) to console and alert.
pub fn send<T>(t: MessageType, payload: &T)
where
	T: Serialize + Debug,
{
	util::with_logging(|| {
		let mut enc = common::Encoder::new(Vec::new());
		encode_msg(&mut enc, t, payload)?;
		Connection::dispatcher().send(Request::Send(enc.finish()?));
		Ok(())
	});
}

/// States of the connection finite state machine
#[derive(Eq, PartialEq, Copy, Clone, Debug)]
pub enum State {
	Loading,
	Connecting,
	Handshaking,
	HandshakeComplete,
	Dropped,
}

impl Default for State {
	fn default() -> Self {
		Self::Loading
	}
}

/// Agent controlling global websocket connection
pub struct Connection {
	/// Link to any subscribers
	link: AgentLink<Self>,

	/// Connection state machine
	state: State,

	/// Connection currently authenticated with
	authed_with: Option<uuid::Uuid>,

	/// Link to global application state
	#[allow(unused)]
	bridge: state::HookBridge,

	/// Reconnection attempts since last connect, if any
	reconn_attempts: i32,

	/// Reconnection timer
	reconn_timer: Option<TimeoutTask>,

	/// Connection to server
	socket: Option<web_sys::WebSocket>,

	/// Active subscribers to connection state change
	subscribers: HashSet<HandlerId>,

	/// Messages deferred till after handshake completion
	deferred: Vec<Vec<u8>>,
}

pub enum Event {
	Open,
	Close(web_sys::CloseEvent),
	Error(web_sys::ErrorEvent),
	Receive(web_sys::MessageEvent),

	TryReconnecting,
	VisibilityChanged,
	WentOnline,
	WentOffline,

	KeyPairChanged,
}

/// Request to send a message
pub enum Request {
	/// Send a regular message
	Send(Vec<u8>),

	/// Send a handshake message
	Handshake {
		/// Send key used to generate message to prevent async race conditions
		key_pair: KeyPair,

		/// Message to send
		message: Vec<u8>,
	},
}

impl Agent for Connection {
	type Reach = Context<Self>;
	type Message = Event;
	type Input = Request;
	type Output = State;

	fn create(link: AgentLink<Self>) -> Self {
		use state::Change;

		let mut s = Self {
			bridge: state::hook(&link, vec![Change::KeyPair], |_| {
				Event::KeyPairChanged
			}),
			authed_with: None,
			link,
			state: State::Loading,
			reconn_attempts: 0,
			reconn_timer: None,
			socket: None,
			subscribers: HashSet::new(),
			deferred: vec![],
		};

		s.connect();

		#[rustfmt::skip]
		macro_rules! bind {
			($target:ident, $event:expr, $variant:ident) => {
				s.add_listener(
					&util::$target(),
					$event,
					|_: web_sys::Event| Event::$variant,
				);
			};
		}

		// Work around browser slowing down or suspending tabs
		bind!(document, "visibilitychange", VisibilityChanged);

		bind!(window, "online", WentOnline);
		bind!(window, "offline", WentOffline);

		s
	}

	fn update(&mut self, msg: Event) {
		use Event::*;

		match msg {
			Open => {
				self.reset_reconn_attempts();
				Self::send_handshake_req(state::read(|s| s.key_pair.clone()))
			}
			KeyPairChanged => {
				state::read(|s| {
					if match (&s.key_pair.id, &self.authed_with) {
						(Some(new), Some(old)) => new != old,
						_ => false,
					} {
						// Reconnect with new key
						self.connect();
					}
				})
			}
			Close(e) => {
				self.reset_socket_and_timer();
				if e.code() != 1000 && e.reason() != "" {
					if e.reason() == "unknown public key ID" {
						state::Agent::dispatcher()
							.send(state::Request::SetKeyID(None));
					} else {
						util::log_and_alert_error(&e.reason())
					}
				}
				self.handle_disconnect();
			}
			Error(e) => {
				self.reset_socket_and_timer();
				util::log_error(&e.message());
				self.set_state(State::Dropped);
			}
			TryReconnecting => {
				if self.state == State::Dropped {
					self.connect();
				}
			}
			Receive(e) => {
				util::log_error_res(
					self.on_message(
						js_sys::Uint8Array::new(&e.data()).to_vec(),
					),
				);
			}
			VisibilityChanged => {
				if util::document().hidden()
					|| !util::window().navigator().on_line()
				{
					match self.state {
						State::HandshakeComplete => {
							// Ensure still connected, in case the computer went
							// to sleep or hibernate or the mobile browser tab
							// was suspended.

							// TODO: Send "ping" to server
						}
						_ => self.connect(),
					}
				}
			}
			WentOnline => self.connect(),
			WentOffline => self.handle_disconnect(),
		};
	}

	fn connected(&mut self, id: HandlerId) {
		self.subscribers.insert(id);
		self.send_current_state(id);
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove(&id);
	}

	fn handle_input(&mut self, req: Self::Input, _: HandlerId) {
		util::with_logging(|| {
			match req {
				Request::Send(msg) => {
					self.send(msg, false)?;
				}
				Request::Handshake { key_pair, message } => {
					// Prevent async race conditions on key pair change
					if state::read(|s| s.key_pair != key_pair) {
						return Ok(());
					}

					self.send(message, true)?;

					// Set state here, because the handshake message is
					// generated async and thus does not have access to self
					self.set_state(State::Handshaking);
				}
			};
			Ok(())
		})
	}
}

impl Connection {
	/// Set new state and send it to all subscribers
	fn set_state(&mut self, new: State) {
		if self.state != new {
			self.state = new;
			for id in self.subscribers.iter() {
				self.send_current_state(*id);
			}
		}
	}

	fn send_current_state(&self, subscriber: HandlerId) {
		self.link.respond(subscriber, self.state)
	}

	fn handle_disconnect(&mut self) {
		self.reconn_attempts += 1;
		self.reconn_timer = Some(TimeoutService::spawn(
			// Maxes out at ~1min
			std::time::Duration::from_millis(
				(500f32
					* 1.5f32.powi(std::cmp::min(self.reconn_attempts / 2, 12)))
					as u64,
			),
			self.link.callback(|_| Event::TryReconnecting),
		));

		self.set_state(State::Dropped);
	}

	fn close_socket(&mut self) {
		if let Some(s) = &self.socket {
			util::log_error_res(s.close());
		}
		self.socket = None;
		self.authed_with = None;
	}

	fn reset_reconn_attempts(&mut self) {
		self.reset_reconn_timer();
		self.reconn_attempts = 0;
	}

	fn reset_reconn_timer(&mut self) {
		self.reconn_timer = None;
	}

	fn reset_socket_and_timer(&mut self) {
		self.close_socket();
		self.reset_reconn_timer();
	}

	fn send(&mut self, mut msg: Vec<u8>, is_handshake: bool) -> util::Result {
		use State::*;

		match (
			match self.state {
				Connecting | Handshaking => is_handshake,
				HandshakeComplete => true,
				_ => false,
			},
			self.socket.as_ref(),
		) {
			(true, Some(soc)) => {
				soc.send_with_u8_array(&mut msg)?;
			}
			_ => {
				self.deferred.push(msg);
			}
		}
		Ok(())
	}

	fn add_listener<T, E, F>(&self, target: &T, event: &str, mapper: F)
	where
		T: AsRef<web_sys::EventTarget>,
		E: wasm_bindgen::convert::FromWasmAbi + 'static,
		F: Fn(E) -> Event + 'static,
	{
		util::add_static_listener(
			target,
			event,
			true,
			self.link.callback(mapper),
		);
	}

	fn connect(&mut self) {
		self.close_socket();
		if !util::window().navigator().on_line() {
			return;
		}

		match || -> util::Result<web_sys::WebSocket> {
			let socket = web_sys::WebSocket::new({
				let loc = util::window().location();
				&format!(
					"{}://{}/api/socket",
					{
						let p = loc.protocol().unwrap();
						match p.as_str() {
							"https:" => "wss",
							"http:" => "ws",
							_ => {
								return Err(format!(
									"unsupported protocol: {}",
									p
								)
								.into());
							}
						}
					},
					loc.host().unwrap(),
				)
			})?;

			socket.set_binary_type(web_sys::BinaryType::Arraybuffer);

			self.add_listener(&socket, "open", |_: web_sys::Event| Event::Open);

			#[rustfmt::skip]
			macro_rules! bind {
				($event:ident, $type:expr, $variant:ident) => {
					self.add_listener(&socket, $type, |e: web_sys::$event| {
						Event::$variant(e)
					});
				};
			}

			bind!(CloseEvent, "close", Close);
			bind!(ErrorEvent, "error", Error);
			bind!(MessageEvent, "message", Receive);

			Ok(socket)
		}() {
			Ok(s) => {
				self.set_state(State::Connecting);
				self.socket = Some(s);
				self.reset_reconn_timer();
			}
			Err(e) => {
				ConsoleService::error(e.as_ref());
			}
		};
	}

	fn on_message(&mut self, data: Vec<u8>) -> util::Result {
		fn decode<T>(t: MessageType, dec: &mut Decoder) -> util::Result<T>
		where
			T: for<'de> serde::Deserialize<'de> + std::fmt::Debug,
		{
			let payload: T = dec.read_next()?;
			debug_log!(format!(">>> {:?}", t), payload);
			Ok(payload)
		}

		let mut dec = Decoder::new(&data)?;

		// TODO: thread page and meta handlers
		// TODO: index thread handler

		while let Some(t) = dec.peek_type() {
			use common::payloads::HandshakeRes;
			use state::Request;
			use MessageType::*;

			macro_rules! decode {
				() => {
					decode(t, &mut dec)?
				};
			}

			macro_rules! decode_empty {
				() => {
					decode!() as ();
				};
			}

			macro_rules! error {
				($err:expr) => {
					return Err($err.into())
				};
				($fmt:literal, $($arg:expr),*$(,)?) => {
					return Err(
						format!($fmt, $($arg)*).into()
					)
				};
			}

			// Send a request to the app state agent
			fn send(req: Request) {
				state::Agent::dispatcher().send(req);
			}

			match t {
				Handshake => {
					let req: HandshakeRes = decode!();
					self.authed_with = Some(req.id.clone());
					if state::read(|s| s.key_pair.id.is_none()) {
						state::Agent::dispatcher().send(
							state::Request::SetKeyID(req.id.clone().into()),
						);
					}

					if req.need_resend {
						// Key already saved in database. Need to confirm
						// it's the same private key by sending a
						// HandshakeReq with Authentication::Saved.
						let mut kp = state::read(|s| s.key_pair.clone());
						kp.id = req.id.into();
						Self::send_handshake_req(kp);
					} else {
						util::with_logging(|| {
							self.set_state(State::HandshakeComplete);
							for msg in std::mem::take(&mut self.deferred) {
								self.send(msg, false)?;
							}
							Ok(())
						});
					}
				}
				InsertThreadAck => {
					let id: u64 = decode!();
					send(Request::SetMine(id));
					send(Request::SetOpenPostID(id.into()));
					state::navigate_to(state::Location {
						feed: state::FeedID::Thread { id, page: 0 },
						focus: None,
					});
				}
				InsertThread => {
					send(Request::InsertThread(decode!()));
				}
				PartitionedPageStart => {
					decode_empty!();
					let mut posts = Vec::<common::payloads::Post>::new();
					loop {
						match dec.peek_type() {
							Some(Post) => {
								posts.push(decode(Post, &mut dec)?);
							}
							Some(PartitionedPageEnd) => {
								decode(PartitionedPageEnd, &mut dec)? as ();
								send(Request::RegisterPage(posts));
								break;
							}
							Some(t @ _) => error!(
								"unexpected message in page stream: {:?}",
								t
							),
							None => {
								error!("incomplete partitioned page stream")
							}
						}
					}
				}
				ThreadMeta => {
					send(Request::RegisterThread(decode!()));
				}
				PartitionedThreadIndexStart => {
					decode_empty!();
					let mut threads =
						Vec::<common::payloads::ThreadWithPosts>::new();
					loop {
						match dec.peek_type() {
							Some(ThreadAbbreviated) => {
								threads
									.push(decode(ThreadAbbreviated, &mut dec)?);
							}
							Some(PartitionedThreadIndexEnd) => {
								decode(PartitionedThreadIndexEnd, &mut dec)?
									as ();
								send(Request::RegisterThreads(threads));
								break;
							}
							Some(t @ _) => error!(
								"unexpected message in thread stream: {:?}",
								t
							),
							None => {
								error!("incomplete partitioned thread stream")
							}
						}
					}
				}
				_ => error!("unhandled message type: {:?}", t),
			}
		}

		Ok(())
	}

	/// Asynchronously generate and send a handshake request message
	fn send_handshake_req(key_pair: KeyPair) {
		use common::payloads::Authorization;

		async fn inner(key_pair: KeyPair) -> util::Result {
			let crypto = util::window().crypto()?;
			let mut enc = common::Encoder::new(Vec::new());
			encode_msg(
				&mut enc,
				MessageType::Handshake,
				&common::payloads::HandshakeReq {
					protocol_version: common::VERSION,
					auth: match &key_pair.id {
						Some(id) => {
							let mut nonce: [u8; 32] = unsafe {
								std::mem::MaybeUninit::uninit().assume_init()
							};
							crypto
								.get_random_values_with_u8_array(&mut nonce)?;

							Authorization::Saved {
								id: id.clone(),
								nonce,
								signature: key_pair
									.sign(&mut {
										let mut buf =
											Vec::with_capacity(16 + 32);
										buf.extend(id.as_bytes());
										buf.extend(&nonce);
										buf
									})
									.await?,
							}
						}
						None => {
							Authorization::NewPubKey(key_pair.public.clone())
						}
					},
				},
			)?;

			Connection::dispatcher().send(Request::Handshake {
				key_pair,
				message: enc.finish()?,
			});

			Ok(())
		}

		wasm_bindgen_futures::spawn_local(util::with_logging_async(
			inner, key_pair,
		));
	}
}

pub struct SyncCounter {
	/// Ensures connection agent is never dropped
	#[allow(unused)]
	conn: Box<dyn Bridge<Connection>>,

	current: State,
}

impl Component for SyncCounter {
	comp_no_props! {}
	type Message = State;

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			conn: Connection::bridge(link.callback(|s| s)),
			current: State::Loading,
		}
	}

	fn update(&mut self, new: State) -> bool {
		self.current = new;
		true
	}

	fn view(&self) -> Html {
		use State::*;

		html! {
			<b id="sync" class="banner-float" title=localize!("sync")>
				{
					localize! {
						match self.current {
							Loading => "loading",
							Connecting | Handshaking => "connecting",
							HandshakeComplete => "connected",
							Dropped => "disconnected",
						}
					}
				}
			</b>
		}
	}
}
