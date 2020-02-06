use super::{state, util};
use protocol::*;
use serde::{Deserialize, Serialize};
use std::collections::HashSet;
use yew::agent::{Agent, AgentLink, Context, HandlerId};
use yew::services::console::ConsoleService;
use yew::services::timeout::{TimeoutService, TimeoutTask};
use yew::services::Task;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

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
}

#[derive(Serialize, Deserialize)]
pub enum Request {
	CurrentState,

	// Send encoded message or message batch to server.
	// Use encode_batch! to encode a message batch.
	SendMsg(Vec<u8>),
}

impl Agent for Connection {
	type Reach = Context;
	type Message = Event;
	type Input = Request;
	type Output = State;

	fn create(link: AgentLink<Self>) -> Self {
		let mut s = Self {
			link: link,
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
				util::log_error_res(|| -> util::Result {
					self.send(state::with(|s| -> util::Result<Vec<u8>> {
						super::encode_batch!(
							MessageType::Handshake,
							&Handshake {
								protocol_version: VERSION,
								key: s.auth_key.clone(),
							},
							MessageType::Synchronize,
							&s.thread
						)
					})?)?;
					self.set_state(State::Syncing);
					Ok(())
				}());
			}
			Event::Close(e) => {
				self.reset_socket_and_timer();
				if e.code() != 1000 {
					util::log_error(e.reason());
					util::window()
						.alert_with_message(&format!("error: {}", e.reason()))
						.expect("alert failed");
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
		};
	}

	fn connected(&mut self, id: HandlerId) {
		self.subscribers.insert(id);
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove(&id);
	}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		match req {
			Request::CurrentState => {
				self.link.respond(id, self.state);
			}
			Request::SendMsg(msg) => {
				util::log_error_res(self.send(msg));
			}
		}
	}
}

impl Connection {
	// Set new state and send it to all subscribers
	fn set_state(&mut self, new: State) {
		self.state = new;
		for id in self.subscribers.iter() {
			self.link.respond(*id, self.state)
		}
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
		let mut dec = Decoder::new(&data)?;
		loop {
			match dec.peek_type() {
				None => return Ok(()),
				Some(t) => match t {
					MessageType::Synchronize => {
						state::with(|s| -> std::io::Result<()> {
							s.thread = dec.read_next()?;
							Ok(())
						})?;
						self.set_state(State::Synced);
					}
					MessageType::FeedInit => {
						// TODO: Use it
						dec.skip_next();
					}
					_ => {
						return Err(util::Error::new(format!(
							"unhandled message type: {:?}",
							t
						)))
					}
				},
			};
		}
	}
}

pub struct SyncCounter {
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

	fn mounted(&mut self) -> bool {
		self.conn.send(Request::CurrentState);
		false
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
					title={localize!("sync")}
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

// Encode a batch of protocol::MessageType and Serialize pairs
#[macro_export]
macro_rules! encode_batch {
	($($type:expr, $payload:expr),+) => {{
		let mut enc =  protocol::Encoder::new(Vec::new());
		$(
			enc.write_message($type, $payload)?;
		)+
		enc.finish().map_err(|e| e.into())
	}};
}
