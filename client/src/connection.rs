use super::{
	state::{self, KeyPair},
	util,
};
use common::{Decoder, Encoder, MessageType};
use serde::Serialize;
use std::{collections::HashSet, fmt::Debug};
use yew::{
	agent::{Agent, AgentLink, Context, Dispatched, HandlerId},
	html, Bridge, Bridged, Component, ComponentLink, Html,
};
use yew_services::timeout::{TimeoutService, TimeoutTask};

// TODO: break up into submodules

// TODO: send open post reclamation request with a full text body for any open
// post on reconnect (via notification). Server-side it should be handled as
// Client -> ThreadFeed -> Client -> websocket response (confirmation or
// failure)

/// Encode message and log it in debug mode
pub fn encode_msg<T>(
	enc: &mut Encoder,
	t: MessageType,
	payload: &T,
) -> std::io::Result<()>
where
	T: Serialize + Debug,
{
	common::log_msg_out!(t, payload);
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
		Connection::dispatcher().send(Request::Send {
			is_open_post_manipulation: matches!(
				t,
				MessageType::Append
					| MessageType::Backspace
					| MessageType::PatchPostBody
					| MessageType::InsertImage
			),
			message: enc.finish()?,
		});
		Ok(())
	});
}

/// States of the connection finite state machine
#[derive(Eq, PartialEq, Copy, Clone, Debug)]
pub enum State {
	/// Module loading
	Loading,

	/// Connecting to server
	Connecting,

	/// Handshake with server in progress
	Handshaking,

	/// Handshake complete - can send regular messages
	HandshakeComplete,

	/// Connection loss
	Disconnected,

	/// Server disconnected client with a critical error. This should mean a
	/// programming error of some sort.
	CriticalError,
}

impl Default for State {
	#[inline]
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
	app_state: state::StateBridge,

	/// Reconnection attempts since last connect, if any
	reconn_attempts: i32,

	/// Reconnection timer
	reconn_timer: Option<TimeoutTask>,

	/// Connection to server
	socket: Option<web_sys::WebSocket>,

	/// Socket handler closures to be freed on socket closure
	handler_closures: Vec<Box<dyn Drop>>,

	/// Active subscribers to connection state change
	subscribers: HashSet<HandlerId>,

	/// Messages deferred till after handshake completion
	deferred: Vec<Vec<u8>>,
}

#[derive(Debug)]
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
	Send {
		/// Messages manipulates an open post and should not be buffered.
		/// This is to guarantee sequentiality of state updates on the server.
		/// The sender is responsible for restoring the sequentiality on
		/// reconnection.
		is_open_post_manipulation: bool,

		/// Message to send
		message: Vec<u8>,
	},

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

	#[cold]
	fn create(link: AgentLink<Self>) -> Self {
		use state::Change;

		let mut s = Self {
			app_state: state::hook(&link, vec![Change::KeyPair], || {
				Event::KeyPairChanged
			}),
			authed_with: None,
			link,
			state: State::Loading,
			reconn_attempts: 0,
			reconn_timer: None,
			socket: None,
			handler_closures: Default::default(),
			subscribers: HashSet::new(),
			deferred: vec![],
		};

		s.connect();

		#[rustfmt::skip]
		macro_rules! bind {
			($target:ident, $event:expr, $variant:ident) => {
				util::add_static_listener(
					&util::$target(),
					$event,
					true,
					s.link.callback(|_: web_sys::Event| Event::$variant,),
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
				Self::send_handshake_req(self.app_state.get().key_pair.clone())
			}
			KeyPairChanged => {
				if match (&self.app_state.get().key_pair.id, &self.authed_with)
				{
					(Some(new), Some(old)) => new != old,
					_ => false,
				} {
					// Reconnect with new key
					self.connect();
				}
			}
			Close(e) => {
				let r = e.reason();
				if e.code() != 1000 && !r.is_empty() {
					if r == "unknown public key ID" {
						state::Agent::dispatcher()
							.send(state::Request::SetKeyID(None));
					} else {
						util::log_and_alert_error(&r);
						self.set_state(State::CriticalError);
						return;
					}
				}
				self.reset_socket_and_timer();
				self.handle_disconnect();
			}
			Error(_) => {
				self.reset_socket_and_timer();
				self.set_state(State::Disconnected);
			}
			TryReconnecting => {
				if self.state == State::Disconnected {
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

	#[cold]
	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove(&id);
	}

	fn handle_input(&mut self, req: Self::Input, _: HandlerId) {
		util::with_logging(|| {
			match req {
				Request::Send {
					message,
					is_open_post_manipulation,
				} => {
					self.send(message, false, !is_open_post_manipulation)?;
				}
				Request::Handshake { key_pair, message } => {
					// Prevent async race conditions on key pair change
					if self.app_state.get().key_pair != key_pair {
						return Ok(());
					}

					self.send(message, true, false)?;

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

	#[inline]
	fn send_current_state(&self, subscriber: HandlerId) {
		self.link.respond(subscriber, self.state)
	}

	#[cold]
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

		self.set_state(State::Disconnected);
	}

	#[cold]
	fn close_socket(&mut self) {
		if let Some(s) = &self.socket {
			util::log_error_res(s.close());
		}
		self.socket = None;

		if !self.handler_closures.is_empty() {
			// Free any closures only after nothing is surely using them
			// anymore. Better than leaking them.
			let cls = std::mem::take(&mut self.handler_closures);
			gloo::timers::callback::Timeout::new(1000 * 60, move || {
				std::mem::drop(cls)
			})
			.forget();
		}

		self.authed_with = None;
	}

	fn reset_reconn_attempts(&mut self) {
		self.reset_reconn_timer();
		self.reconn_attempts = 0;
	}

	#[inline]
	fn reset_reconn_timer(&mut self) {
		self.reconn_timer = None;
	}

	fn reset_socket_and_timer(&mut self) {
		self.close_socket();
		self.reset_reconn_timer();
	}

	fn send(
		&mut self,
		mut msg: Vec<u8>,
		is_handshake: bool,
		defer: bool,
	) -> util::Result {
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
				if defer {
					self.deferred.push(msg);
				}
			}
		}
		Ok(())
	}

	#[cold]
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

			macro_rules! add_listener {
				($event:expr, $mapper:expr) => {
					self.handler_closures.push(Box::new(util::add_listener(
						&socket,
						$event,
						true,
						self.link.callback($mapper),
					)))
				};
				($event:expr, $web_sys_event:ident, $variant:ident) => {
					add_listener!($event, |e: web_sys::$web_sys_event| {
						Event::$variant(e)
					});
				};
			}

			add_listener!("open", |_: web_sys::Event| Event::Open);
			add_listener!("close", CloseEvent, Close);
			add_listener!("error", ErrorEvent, Error);
			add_listener!("message", MessageEvent, Receive);

			Ok(socket)
		}() {
			Ok(s) => {
				self.set_state(State::Connecting);
				self.socket = Some(s);
				self.reset_reconn_timer();
			}
			Err(e) => {
				util::log_error(&e);
			}
		};
	}

	fn on_message(&mut self, data: Vec<u8>) -> util::Result {
		#[inline]
		fn decode<T>(t: MessageType, dec: &mut Decoder) -> util::Result<T>
		where
			T: for<'de> serde::Deserialize<'de> + std::fmt::Debug,
		{
			let payload: T = dec.read_next()?;
			common::log_msg_in!(t, payload);
			Ok(payload)
		}

		let mut dec = Decoder::new(&data)?;

		while let Some(t) = dec.peek_type() {
			use common::payloads::{HandshakeRes, PubKeyStatus};
			use state::Request;
			use MessageType::*;

			macro_rules! decode {
				($t:expr) => {
					decode($t, &mut dec)?
				};
				() => {
					decode!(t)
				};
			}

			macro_rules! skip_payload {
				($t:expr) => {
					dec.skip_next();
					common::log_msg_in!($t, ());
				};
				() => {
					skip_payload!(t);
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
			#[inline]
			fn send(req: Request) {
				state::Agent::dispatcher().send(req);
			}

			/// Handle post insertion acknowledgement by the server
			fn ack_post_insertion(
				dec: &mut Decoder,
				t: MessageType,
			) -> util::Result<u64> {
				let id: u64 = decode(t, dec)?;
				send(Request::SetMine(id));

				// Insert a placeholder post, so the postform has something
				// to render
				send(Request::RegisterPost(common::payloads::Post::new(
					id,
					id,
					0,
					util::now(),
					Default::default(),
				)));
				send(Request::SetOpenPostID(id.into()));

				Ok(id)
			}

			match t {
				Handshake => {
					let req: HandshakeRes = decode!();
					self.authed_with = Some(req.id.clone());
					if self.app_state.get().key_pair.id.is_none() {
						state::Agent::dispatcher().send(
							state::Request::SetKeyID(req.id.clone().into()),
						);
					}

					match req.status {
						PubKeyStatus::Accepted => {
							util::with_logging(|| {
								self.set_state(State::HandshakeComplete);
								for msg in std::mem::take(&mut self.deferred) {
									self.send(msg, false, false)?;
								}
								Ok(())
							});
						}
						PubKeyStatus::NeedResend => {
							// Key already saved in database. Need to confirm
							// it's the same private key by sending a
							// HandshakeReq with Authentication::Saved.
							let mut kp = self.app_state.get().key_pair.clone();
							kp.id = req.id.into();
							Self::send_handshake_req(kp);
						}
						PubKeyStatus::NotFound => {
							send(Request::SetKeyID(None));
							let mut kp = self.app_state.get().key_pair.clone();
							kp.id = req.id.into();
							Self::send_handshake_req(kp);
						}
					};
				}
				InsertThreadAck => {
					let id = ack_post_insertion(&mut dec, t)?;
					state::navigate_to(state::Location {
						feed: state::FeedID::Thread { id, page: 0 },
						focus: None,
					});
				}
				InsertPostAck => {
					ack_post_insertion(&mut dec, t)?;
				}
				InsertThread => send(Request::RegisterThread(decode!())),
				InsertPost => {
					let msg: common::payloads::PostCreationNotification =
						decode!();
					send(Request::RegisterPost(common::payloads::Post::new(
						msg.id, msg.thread, msg.page, msg.time, msg.opts,
					)));
				}
				PatchPostBody => send(Request::PatchPostBody(decode!())),
				ClosePost => send(Request::ClosePost(decode!())),
				PartitionedPageStart => {
					skip_payload!();
					let mut posts = Vec::<common::payloads::Post>::new();
					loop {
						match dec.peek_type() {
							Some(Post) => {
								posts.push(decode!(Post));
							}
							Some(PartitionedPageEnd) => {
								skip_payload!(PartitionedPageEnd);
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
				ThreadMeta => send(Request::RegisterThreadMeta(decode!())),
				PartitionedThreadIndexStart => {
					skip_payload!();
					let mut threads =
						Vec::<common::payloads::ThreadWithPosts>::new();
					loop {
						match dec.peek_type() {
							Some(ThreadAbbreviated) => {
								threads.push(decode!(ThreadAbbreviated));
							}
							Some(PartitionedThreadIndexEnd) => {
								skip_payload!(PartitionedThreadIndexEnd);
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
				UsedTags => send(Request::SetUsedTags(decode!())),
				CurrentTime => {
					let server_time: u32 = decode!();
					let now = (js_sys::Date::now() / 1000_f64) as i64;
					send(Request::SetTimeCorrection(
						(server_time as i64 - now) as i32,
					));
				}
				Configs => send(Request::SetConfigs(decode!())),
				_ => error!("unhandled message type: {:?}", t),
			}
		}

		Ok(())
	}

	/// Asynchronously generate and send a handshake request message
	#[cold]
	fn send_handshake_req(key_pair: KeyPair) {
		use common::payloads::Authorization;

		wasm_bindgen_futures::spawn_local(util::with_logging_async(
			async move {
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
									std::mem::MaybeUninit::uninit()
										.assume_init()
								};
								crypto.get_random_values_with_u8_array(
									&mut nonce,
								)?;

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
							None => Authorization::NewPubKey(
								key_pair.public.clone(),
							),
						},
					},
				)?;

				Connection::dispatcher().send(Request::Handshake {
					key_pair,
					message: enc.finish()?,
				});

				Ok(())
			},
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

	#[cold]
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

		let mut cls = vec!["banner-float"];
		if self.current == State::CriticalError {
			cls.push("admin");
		}

		html! {
			<b id="sync" class=cls title=localize!("sync")>
				{
					localize! {
						match self.current {
							Loading => "loading",
							Connecting | Handshaking => "connecting",
							HandshakeComplete => "connected",
							Disconnected => "disconnected",
							CriticalError => "critical_error",
						}
					}
				}
			</b>
		}
	}
}
