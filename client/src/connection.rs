use super::fsm::FSM;
use super::state;
use super::state::State;
use super::util;
use std::sync::Once;
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use web_sys::{CloseEvent, ErrorEvent, MessageEvent};

super::gen_global!(FSM<ConnState, ConnEvent>, FSM::new(ConnState::Loading));

// Websocket connection and synchronization with server states
enum SyncStatus {
	Disconnected,
	Connecting,
	Syncing,
	Synced,
	Desynced,
}

// States of the connection finite state machine
#[repr(u8)]
#[derive(Eq, PartialEq, Hash, Copy, Clone)]
enum ConnState {
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
enum ConnEvent {
	Start,
	Open,
	Close,
	Retry,
	Error,
	Sync,
}

fn render_status(s: ConnState) {
	// TODO: Actually render the status according to language pack
}

fn close_socket(s: &mut State) {
	if let Some(s) = &s.socket {
		s.close();
	}
	s.socket = None;
}

fn on_open() {
	state::with(|s| with(|c| c.feed(s, ConnEvent::Open)));
}

fn on_close(e: CloseEvent) {
	state::with(|s| with(|c| c.feed(s, ConnEvent::Open)));
}

fn on_error(e: ErrorEvent) {
	state::with(|s| with(|c| c.feed(s, ConnEvent::Open)));
}

fn on_message(e: MessageEvent) {
	state::with(|s| {
		with(|c| {
			unimplemented!();
		});
	});
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
	.expect("could not create websocket instance");

	static mut ON_OPEN: Option<Closure<dyn Fn()>> = None;
	static mut ON_CLOSE: Option<Closure<dyn Fn(CloseEvent)>> = None;
	static mut ON_ERROR: Option<Closure<dyn Fn(ErrorEvent)>> = None;
	static mut ON_MESSAGE: Option<Closure<dyn Fn(MessageEvent)>> = None;
	static ONCE: Once = Once::new();
	ONCE.call_once(|| {
		macro_rules! wrap {
			($var:ident, $fn:ident) => {
				unsafe { $var = Some(Closure::wrap(Box::new(&$fn))) };
			};
		}

		wrap!(ON_OPEN, on_open);
		wrap!(ON_CLOSE, on_close);
		wrap!(ON_ERROR, on_error);
		wrap!(ON_MESSAGE, on_message);
	});

	macro_rules! set {
		($prop:ident, $var:ident) => {
			socket.$prop(Some(unsafe {
				$var.as_ref().unwrap().as_ref().unchecked_ref()
			}));
		};
	}

	set!(set_onopen, ON_OPEN);
	set!(set_onclose, ON_CLOSE);
	set!(set_onerror, ON_ERROR);
	set!(set_onerror, ON_MESSAGE);

	s.socket = Some(socket);
}

// Initiate websocket connection to server
pub fn start(state: &mut State) {
	with(|c| {
		c.on_change(&|_, s| render_status(s));

		c.set_transitions(
			&[ConnState::Loading],
			&[ConnEvent::Start],
			&|s, _, _| {
				s.reconn_attempts = 0;
				connect(s);
				ConnState::Connecting
			},
		);
		c.set_transitions(
			&[ConnState::Connecting, ConnState::Reconnecting],
			&[ConnEvent::Sync],
			&|_, _, _| ConnState::Synced,
		);
		c.set_transitions(
			&[ConnState::Syncing],
			&[ConnEvent::Sync],
			&|_, _, _| ConnState::Synced,
		);
		// c.set_any_state_transitions(&[ConnEvent::Close], );

		c.feed(state, ConnEvent::Start);
	});
}
