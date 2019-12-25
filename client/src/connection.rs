use super::cache_cb;
use super::event_handler;
use super::fsm::FSM;
use super::state;
use super::state::State;
use super::util;
use wasm_bindgen::prelude::*;
use web_sys::{CloseEvent, ErrorEvent, MessageEvent};

super::gen_global!(FSM<ConnState, ConnEvent>, FSM::new(ConnState::Loading));

// States of the connection finite state machine
#[repr(u8)]
#[derive(Eq, PartialEq, Hash, Copy, Clone)]
pub enum ConnState {
	Loading,
	Connecting,
	Syncing,
	Synced,
	Reconnecting,
	Dropped,
	Desynced,
}

// Events passable to the connection FSM
#[repr(u8)]
#[derive(Eq, PartialEq, Hash, Copy, Clone)]
pub enum ConnEvent {
	Start,
	Open,
	Close,
	Retry,
	Error,
	Sync,
}

fn render_status(s: ConnState) -> util::Result {
	super::cache_el!("sync").set_text_content(Some(super::localize! {
		match s {
			ConnState::Loading => "loading",
			ConnState::Connecting => "connecting",
			ConnState::Desynced => "desynced",
			ConnState::Dropped => "disconnected",
			ConnState::Reconnecting => "connecting",
			ConnState::Synced => "synced",
			ConnState::Syncing => "syncing",
		}
	}));
	Ok(())
}

fn close_socket(s: &mut State) {
	if let Some(s) = &s.conn.socket {
		s.close().expect("failed to closed socket");
	}
	s.conn.socket = None;
}

// Run closure with application and connection state as arguments
fn with_state<F, R>(f: F) -> R
where
	F: Fn(&mut State, &mut FSM<ConnState, ConnEvent>) -> R,
{
	state::with(|s| with(|c| f(s, c)))
}

// Encode a batch of protocol::MessageType and Serialize pairs
#[macro_export]
macro_rules! encode_batch {
	($($type:expr, $payload:expr),+) => {{
		let mut enc =  protocol::Encoder::new(Vec::new());
		$(
			enc.write_message($type, $payload)?;
		)+
		&mut enc.finish()?
	}};
}

// Send message batch to server. Use encode_batch! to encode a message batch.
pub fn send(s: &mut State, payload: &mut [u8]) -> util::Result {
	if let Some(ref soc) = s.conn.socket {
		soc.send_with_u8_array(payload)?;
	}
	Ok(())
}

fn connect(s: &mut State) {
	close_socket(s);

	let socket = web_sys::WebSocket::new({
		let loc = util::window().location();
		&format!(
			"{}://{}/api/socket",
			{
				let p = loc.protocol().expect("could not get protocol");
				match p.as_str() {
					"https:" => "wss",
					"http:" => "ws",
					"file:" => {
						panic!("page downloaded locally; refusing to sync")
					}
					_ => panic!(format!("unknown protocol: {}", p)),
				}
			},
			loc.host().expect("could not get host"),
		)
	})
	.expect("could not open websocket connection");

	socket.set_binary_type(web_sys::BinaryType::Arraybuffer);

	macro_rules! set {
		($prop:ident, $type:ty, $fn:expr) => {
			socket.$prop(Some(cache_cb!($type, $fn)));
		};
	}

	set!(set_onopen, dyn Fn(), || {
		with_state(|s, c| {
			util::log_error_res(c.feed(s, ConnEvent::Open));
		})
	});
	set!(set_onclose, dyn Fn(CloseEvent), |e: CloseEvent| {
		with_state(|s, c| {
			if e.code() != 1000 {
				util::log_error(e.reason());
			}
			util::log_error_res(c.feed(s, ConnEvent::Close));
		})
	});
	set!(set_onerror, dyn Fn(ErrorEvent), |e: ErrorEvent| {
		with_state(|s, c| {
			util::log_error(e.message());
			util::log_error_res(c.feed(s, ConnEvent::Error));
		});
	});
	set!(set_onmessage, dyn Fn(MessageEvent), |e: MessageEvent| {
		with_state(|s, c| {
			util::log_error_res(on_message(s, c, e.data()));
		});
	});

	s.conn.socket = Some(socket);
}

fn on_message(
	s: &mut State,
	c: &mut FSM<ConnState, ConnEvent>,
	data: JsValue,
) -> util::Result {
	use protocol::*;

	let mut dec = Decoder::new(&js_sys::Uint8Array::new(&data).to_vec())?;
	loop {
		match dec.peek_type() {
			None => return Ok(()),
			Some(t) => match t {
				MessageType::Synchronize => {
					s.thread = dec.read_next()?;
					c.feed(s, ConnEvent::Sync)?;
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

// Reset module state
fn reset(s: &mut State) {
	close_socket(s);
	reset_reconn_timer(s);
}

fn reset_reconn_timer(s: &mut State) {
	if s.conn.reconn_timer != 0 {
		util::window().clear_timeout_with_handle(s.conn.reconn_timer);
		s.conn.reconn_timer = 0;
	}
}

fn reset_reconn_attempts(s: &mut State) {
	reset_reconn_timer(s);
	s.conn.reconn_attempts = 0;
}

// Initiate websocket connection to server
pub fn init(state: &mut State) -> util::Result {
	with(|c| {
		c.on_change(&|_, s| render_status(s));

		c.set_transitions(
			&[ConnState::Loading],
			&[ConnEvent::Start],
			&|s, _, _| {
				s.conn.reconn_attempts = 0;
				connect(s);
				Ok(ConnState::Connecting)
			},
		);

		c.set_transitions(
			&[ConnState::Connecting, ConnState::Reconnecting],
			&[ConnEvent::Open],
			&|s, _, _| {
				use protocol::*;

				reset_reconn_attempts(s);
				send(
					s,
					encode_batch!(
						MessageType::Handshake,
						&Handshake {
							protocol_version: VERSION,
							key: s.auth_key.clone(),
						},
						MessageType::Synchronize,
						// TODO: Send actual thread number
						&0u64
					),
				)?;
				Ok(ConnState::Syncing)
			},
		);

		c.set_transitions(
			&[ConnState::Syncing],
			&[ConnEvent::Sync],
			&|_, _, _| Ok(ConnState::Synced),
		);

		c.set_any_state_transitions(&[ConnEvent::Close], &|s, state, _| {
			reset(s);

			s.conn.reconn_attempts += 1;
			util::window()
				.set_timeout_with_callback_and_timeout_and_arguments_0(
					cache_cb!(dyn Fn(), || {
						with_state(|s, c| {
							util::log_error_res(c.feed(s, ConnEvent::Retry));
						});
					}),
					// Maxes out at ~1min
					(500f32
						* 1.5f32.powi(std::cmp::min(
							s.conn.reconn_attempts / 2,
							12,
						))) as i32,
				)
				.unwrap();

			Ok(if state == ConnState::Desynced {
				ConnState::Desynced
			} else {
				ConnState::Dropped
			})
		});

		c.set_transitions(
			&[ConnState::Dropped],
			&[ConnEvent::Retry],
			&|s, _, _| {
				Ok(if util::window().navigator().on_line() {
					connect(s);
					ConnState::Reconnecting
				} else {
					ConnState::Dropped
				})
			},
		);

		c.set_any_state_transitions(&[ConnEvent::Error], &|s, _, _| {
			reset(s);
			Ok(ConnState::Desynced)
		});

		c.feed(state, ConnEvent::Start)?;

		// Work around browser slowing down/suspending tabs and keep the FSM up
		// to date with the actual tab status
		util::add_listener(
			util::document(),
			"visibilitychange",
			event_handler!(|_| {
				with_state(|s, c| {
					if util::document().hidden()
						|| !util::window().navigator().on_line()
					{
						return Ok(());
					}
					match c.state() {
						// Ensure still connected, in case the computer went
						// to sleep or hibernate or the mobile browser tab
						// was suspended
						ConnState::Synced => {
							// TODO: Send ping to server
							Ok(())
						}
						ConnState::Desynced => Ok(()),
						_ => c.feed(s, ConnEvent::Retry),
					}
				})
			}),
		);

		util::add_listener(
			util::window(),
			"online",
			event_handler!(|_| {
				with_state(|s, c| {
					reset_reconn_attempts(s);
					c.feed(s, ConnEvent::Retry)
				})
			}),
		);
		util::add_listener(
			util::window(),
			"offline",
			event_handler!(|_| with_state(|s, c| c.feed(s, ConnEvent::Close))),
		);

		Ok(())
	})
}
