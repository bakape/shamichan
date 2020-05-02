use super::{
	state::{self, KeyPair},
	util,
};
use protocol::{debug_log, Decoder, Encoder, MessageType};
use serde::{Deserialize, Serialize};
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

// Send a message over websocket.
// Log any encoding errors (there should not be any) to console and alert.
pub fn send<T>(t: MessageType, payload: &T)
where
	T: Serialize + Debug,
{
	util::with_logging(|| {
		let mut enc = protocol::Encoder::new(Vec::new());
		encode_msg(&mut enc, t, payload)?;
		Connection::dispatcher().send(Request::Regular(enc.finish()?));
		Ok(())
	});
}

// States of the connection finite state machine
#[derive(Serialize, Deserialize, Eq, PartialEq, Copy, Clone, Debug)]
pub enum State {
	Loading,
	Connecting,
	Handshaking,
	Syncing,
	Synced,
	Dropped,
}

// Agent controlling global websocket connection
pub struct Connection {
	// Link to any subscribers
	link: AgentLink<Self>,

	// Connection state machine
	state: State,

	// Feed currently being synced
	syncing_to: Option<u64>,

	// Connection currently authenticated with
	authed_with: Option<uuid::Uuid>,

	// Link to global application state
	#[allow(unused)]
	bridge: state::HookBridge,

	// Reconnection attempts since last connect, if any
	reconn_attempts: i32,

	// Reconnection timer
	reconn_timer: Option<TimeoutTask>,

	// Connection to server
	socket: Option<web_sys::WebSocket>,

	// Active subscribers to connection state change
	subscribers: HashSet<HandlerId>,
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

	CheckUpdates,
}

#[derive(Serialize, Deserialize)]
pub enum MessageCategory {
	Handshake,
	Synchronize,
	Regular,
}

// Request to send a message
#[derive(Serialize, Deserialize)]
pub enum Request {
	// Send a regular message
	Regular(Vec<u8>),

	// Send a handshake message
	Handshake {
		// Send key used to generate message to prevent async race conditions
		with_key_pair: KeyPair,

		message: Vec<u8>,
	},
}

impl Agent for Connection {
	type Reach = Context;
	type Message = Event;
	type Input = Request;
	type Output = State;

	fn create(link: AgentLink<Self>) -> Self {
		use state::Change;

		let mut s = Self {
			bridge: state::hook(
				&link,
				&[Change::KeyPair, Change::Location],
				|_| Event::CheckUpdates,
			),
			syncing_to: None,
			authed_with: None,
			link,
			state: State::Loading,
			reconn_attempts: 0,
			reconn_timer: None,
			socket: None,
			subscribers: HashSet::new(),
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
		match msg {
			Event::Open => {
				self.reset_reconn_attempts();
				Self::send_handshake_req(state::read(|s| s.key_pair.clone()))
			}
			Event::CheckUpdates => {
				util::log_error_res(state::read(|s| -> util::Result {
					if match (&s.key_pair.id, &self.authed_with) {
						(Some(new), Some(old)) => new != old,
						_ => false,
					} {
						// Reconnect with new key
						self.connect();
						return Ok(());
					}

					let feed_u64 = s.location.feed.as_u64();
					if match &self.syncing_to {
						Some(old) => *old != feed_u64,
						None => match self.state {
							State::Synced | State::Syncing => true,
							_ => false,
						},
					} {
						self.synchronize(feed_u64)?;
					}

					Ok(())
				}))
			}
			Event::Close(e) => {
				self.reset_socket_and_timer();
				if e.code() != 1000 && e.reason() != "" {
					if e.reason() == "unknown public key ID" {
						state::Agent::dispatcher()
							.send(state::Request::SetKeyID(None));
					} else {
						util::log_error(e.reason());
						util::alert(&e.reason());
					}
				}
				self.handle_disconnect();
			}
			Event::Error(e) => {
				self.reset_socket_and_timer();
				util::log_error(&e.message());
				self.set_state(State::Dropped);
			}
			Event::TryReconnecting => {
				if self.state == State::Dropped {
					self.connect();
				}
			}
			Event::Receive(e) => {
				util::log_error_res(
					self.on_message(
						js_sys::Uint8Array::new(&e.data()).to_vec(),
					),
				);
			}
			Event::VisibilityChanged => {
				if util::document().hidden()
					|| !util::window().navigator().on_line()
				{
					match self.state {
						State::Synced => {
							// Ensure still connected, in case the computer went
							// to sleep or hibernate or the mobile browser tab
							// was suspended.

							// TODO: Send ping to server
						}
						_ => self.connect(),
					}
				}
			}
			Event::WentOnline => self.connect(),
			Event::WentOffline => self.handle_disconnect(),
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
				Request::Regular(msg) => {
					self.send(MessageCategory::Regular, msg)?;
				}
				Request::Handshake {
					with_key_pair,
					message,
				} => {
					// Prevent async race conditions on key pair change
					if state::read(|s| s.key_pair != with_key_pair) {
						return Ok(());
					}

					self.send(MessageCategory::Handshake, message)?;

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
	// Set new state and send it to all subscribers
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
		self.reconn_timer = Some(TimeoutService::new().spawn(
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
		self.syncing_to = None;
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

	fn send(&self, cat: MessageCategory, mut msg: Vec<u8>) -> util::Result {
		match (
			match self.state {
				State::Connecting => matches!(cat, MessageCategory::Handshake),
				State::Handshaking => matches!(
					cat,
					MessageCategory::Handshake | MessageCategory::Synchronize
				),
				State::Synced | State::Syncing => true,
				_ => false,
			},
			self.socket.as_ref(),
		) {
			(true, Some(soc)) => {
				soc.send_with_u8_array(&mut msg)?;
			}
			_ => {
				return Err(format!(
					concat!(
						"sending message when connection not ready: ",
						"state={:?} socket_state={}"
					),
					self.state,
					self.socket
						.as_ref()
						.map(|s| s.ready_state() as isize)
						.unwrap_or(-1)
				)
				.into());
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
		util::add_static_listener(target, event, self.link.callback(mapper));
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
				ConsoleService::new().error(e.as_ref());
			}
		};
	}

	fn on_message(&mut self, data: Vec<u8>) -> util::Result {
		// Helper to make message handling through route!() more terse
		struct HandlerResult(util::Result);

		impl From<()> for HandlerResult {
			fn from(_: ()) -> HandlerResult {
				HandlerResult(Ok(()))
			}
		}

		impl From<util::Result> for HandlerResult {
			fn from(v: util::Result) -> HandlerResult {
				HandlerResult(v)
			}
		}

		impl Into<util::Result> for HandlerResult {
			fn into(self) -> util::Result {
				self.0
			}
		}

		// Separate function to enable type inference of payload type from
		// lambda argument type
		fn _route<'de, T, R>(
			dec: &'de mut Decoder,
			typ: MessageType,
			mut handler: impl FnMut(T) -> R,
		) -> util::Result
		where
			T: Deserialize<'de> + Debug,
			R: Into<HandlerResult>,
		{
			let payload: T = dec.read_next()?;
			debug_log!(format!(">>> {:?}", typ), payload);
			(handler(payload).into() as HandlerResult).into()
		}

		let mut dec = Decoder::new(&data)?;

		macro_rules! route {
			($type:expr, $($msg_type:ident => $handler:expr)+) => {
				match $type {
					$(
						MessageType::$msg_type => {
							_route(&mut dec, MessageType::$msg_type, $handler)?
						}
					)+
					_ => {
						return Err(util::Error::new(format!(
							"unhandled message type: {:?}",
							$type
						)))
					}
				}
			};
		}

		loop {
			use protocol::payloads::{
				FeedData, HandshakeRes, ThreadCreationNotice,
			};

			match dec.peek_type() {
				Some(t) => route! { t,
					Synchronize => |id: u64| {
						// Guard against rapid successive feed changes
						if id == state::read(|s| s.location.feed.as_u64()) {
							self.set_state(State::Synced);
						}
					}
					Handshake => |req: HandshakeRes| {
						self.authed_with = Some(req.id.clone());
						if state::read(|s| s.key_pair.id.is_none()) {
							state::Agent::dispatcher()
								.send(state::Request::SetKeyID(
									req
									.id
									.clone()
									.into(),
								));
						}

						if req.need_resend {
							// Key already saved in database. Need to confirm
							// it's the same private key by sending a
							// HandshakeReq with Authentication::Saved.
							let mut kp = state::read(|s| s.key_pair.clone());
							kp.id = req.id.into();
							Self::send_handshake_req(kp);
						} else {
							util::log_error_res(
								self.synchronize(
									state::read(|s| s.location.feed.as_u64()),
								),
							);
						}
					}
					FeedInit => |_: FeedData| {
						// TODO: Patch existing post data with more up to date
						// patch set. The patch set needs to be stored in
						// state::Agent and applied to the data fetch via the
						// JSON API, no matter which request arrives first.
						// Also need to account for data races on feed
						// switching.
					}
					CreateThreadAck => |_: u64| {
						// TODO: Save thread as owned and navigate to it
					}
					CreateThread => |_: ThreadCreationNotice| {
							// TODO: Insert thread into registry and rerender
							// page, if needed
					}
				},
				None => return Ok(()),
			};
		}
	}

	// Asynchronously generate and send a handshake request message
	fn send_handshake_req(key_pair: state::KeyPair) {
		use protocol::payloads::Authorization;

		async fn inner(key_pair: state::KeyPair) -> util::Result {
			let crypto = util::window().crypto()?;
			let mut enc = protocol::Encoder::new(Vec::new());
			encode_msg(
				&mut enc,
				MessageType::Handshake,
				&protocol::payloads::HandshakeReq {
					protocol_version: protocol::VERSION,
					auth: match &key_pair.id {
						Some(id) => {
							let mut nonce: [u8; 32] = Default::default();
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
				with_key_pair: key_pair,
				message: enc.finish()?,
			});

			Ok(())
		}

		wasm_bindgen_futures::spawn_local(util::with_logging_async(
			inner, key_pair,
		));
	}

	// Send request to synchronize with a feed
	fn synchronize(&mut self, feed: u64) -> util::Result {
		let mut enc = protocol::Encoder::new(Vec::new());
		encode_msg(&mut enc, MessageType::Synchronize, &feed)?;
		self.send(MessageCategory::Synchronize, enc.finish()?)?;

		self.set_state(State::Syncing);
		self.syncing_to = feed.into();

		Ok(())
	}
}

pub struct SyncCounter {
	// Ensures connection agent is never dropped
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
		html! {
			<b
				id="sync"
				class="banner-float svg-link"
				title=localize!("sync")
			>
				{
					localize! {
						match self.current {
							State::Loading => "loading",
							State::Connecting => "connecting",
							State::Dropped => "disconnected",
							State::Synced => "synced",
							State::Syncing => "syncing",
							State::Handshaking => "handshaking"
						}
					}
				}
			</b>
		}
	}
}
