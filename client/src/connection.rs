use super::{state, util};
use protocol::*;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use std::fmt::Debug;
use yew::agent::{Agent, AgentLink, Context, HandlerId};
use yew::services::console::ConsoleService;
use yew::services::timeout::{TimeoutService, TimeoutTask};
use yew::services::Task;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

// Encode a batch of (protocol::MessageType, Serialize) pairs
#[macro_export]
macro_rules! encode_message {
	($($type:expr, $payload:expr),+) => {{
		use protocol::debug_log;

		let mut enc =  protocol::Encoder::new(Vec::new());
		$(
			debug_log!(format!("<<< {:?}: {:?}", $type, $payload));
			enc.write_message($type, $payload)?;
		)+
		enc.finish()
	}};
}

// States of the connection finite state machine
#[derive(Serialize, Deserialize, Eq, PartialEq, Copy, Clone)]
pub enum State {
	Loading,
	Connecting,
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

	// Link to global application state
	app_state: Box<dyn Bridge<state::Agent>>,

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

	AuthKeyChanged,

	NOP,
}

impl Agent for Connection {
	type Reach = Context;
	type Message = Event;
	type Input = Vec<u8>;
	type Output = State;

	fn create(link: AgentLink<Self>) -> Self {
		use state::{Request, Subscription};

		let mut s = Self {
			app_state: state::Agent::bridge(link.callback(|u| match u {
				Subscription::AuthKey => Event::AuthKeyChanged,
				_ => Event::NOP,
			})),
			link,
			state: State::Loading,
			reconn_attempts: 0,
			reconn_timer: None,
			socket: None,
			subscribers: HashSet::new(),
		};

		s.app_state.send(Request::Subscribe(Subscription::AuthKey));
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

		// Work around browser slowing down/suspending tabs
		bind!(document, "visibilitychange", VisibilityChanged);

		bind!(window, "online", WentOnline);
		bind!(window, "offline", WentOffline);

		s
	}

	fn update(&mut self, msg: Event) {
		match msg {
			Event::Open => {
				self.reset_reconn_attempts();
				util::with_logging(|| {
					let s = state::get();
					self.send(encode_message!(
						MessageType::Handshake,
						&Handshake {
							protocol_version: VERSION,
							key: s.auth_key.clone(),
						},
						MessageType::Synchronize,
						&s.feed
					)?)?;
					self.set_state(State::Syncing);
					Ok(())
				});
			}
			Event::Close(e) => {
				self.reset_socket_and_timer();
				if e.code() != 1000 && e.reason() != "" {
					util::log_error(e.reason());
					util::alert(&e.reason());
				}
				self.handle_disconnect();
			}
			Event::Error(e) => {
				self.reset_socket_and_timer();
				util::log_error(format!("{:?}", e));
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
			Event::WentOnline => {
				self.connect();
			}
			Event::WentOffline => self.handle_disconnect(),
			Event::AuthKeyChanged => {
				// Reconnect with new key
				self.connect()
			}
			Event::NOP => (),
		};
	}

	fn connected(&mut self, id: HandlerId) {
		self.subscribers.insert(id);
		self.send_current_state(id);
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove(&id);
	}

	fn handle_input(&mut self, msg: Self::Input, _: HandlerId) {
		util::log_error_res(self.send(msg));
	}
}

impl Connection {
	// Set new state and send it to all subscribers
	fn set_state(&mut self, new: State) {
		self.state = new;
		for id in self.subscribers.iter() {
			self.send_current_state(*id);
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
	}

	fn reset_reconn_attempts(&mut self) {
		self.reset_reconn_timer();
		self.reconn_attempts = 0;
	}

	fn reset_reconn_timer(&mut self) {
		if let Some(mut t) = self.reconn_timer.take() {
			t.cancel();
		}
	}

	fn reset_socket_and_timer(&mut self) {
		self.close_socket();
		self.reset_reconn_timer();
	}

	fn send(&mut self, mut msg: Vec<u8>) -> util::Result {
		if let Some(soc) = self.socket.as_ref() {
			soc.send_with_u8_array(&mut msg)?;
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
		// Helper to make message handling through decode!() more terse
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
		fn _decode<'de, T, R>(
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
		macro_rules! decode {
			($type:expr, $($msg_type:ident => $handler:expr)+) => {
				match $type {
					$(
						MessageType::$msg_type => {
							_decode(&mut dec, MessageType::$msg_type, $handler)?
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
			match dec.peek_type() {
				Some(t) => decode! { t,
					Synchronize => |_: u64| {
						// Feed ID should already be set to the new one at this
						// point, if the client was navigating away
						self.set_state(State::Synced);
					}
					FeedInit => |_: FeedData| {
						// TODO: Sync to feed
					}
					CreateThreadAck => |_: u64| {
						// TODO: Save thread as owned and navigate to it
					}
					CreateThread => |_: ThreadCreationNotice| {
						if state::get().feed != 0 {
							// TODO: Insert thread into registry and rerender
							// page, if needed
						}
					}
				},
				None => return Ok(()),
			};
		}
	}
}

pub struct SyncCounter {
	// Ensures connection agent is never dropped
	#[allow(unused)]
	conn: Box<dyn Bridge<Connection>>,

	current: State,
}

impl Component for SyncCounter {
	type Message = State;
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			conn: Connection::bridge(link.callback(|s| s)),
			current: State::Loading,
		}
	}

	fn update(&mut self, new: State) -> bool {
		if self.current != new {
			self.current = new;
			true
		} else {
			false
		}
	}

	fn view(&self) -> Html {
		html! {
			<>
				<b
					id="sync"
					class="banner-float svg-link"
					title=localize!("sync")
				>{
					localize! {
						match self.current {
							State::Loading => "loading",
							State::Connecting => "connecting",
							State::Dropped => "disconnected",
							State::Synced => "synced",
							State::Syncing => "syncing",
						}
					}
				}</b>
			</>
		}
	}
}
