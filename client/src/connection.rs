use super::fsm::FSM;
use super::state;
use super::state::State;
use super::util;
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

	#[rustfmt::skip]
	macro_rules! set {
		($prop:ident,  $fn:ident, $type:ty) => {
			unsafe {
				static mut CACHED: Option<Closure<$type>> = None;
				if CACHED.is_none() {
					CACHED = Some(Closure::wrap(Box::new(&$fn)));
				}
				socket.$prop(
					Some(
						CACHED.as_ref().unwrap().as_ref().unchecked_ref(),
					),
				);
			}
		};
	}

	set!(set_onopen, on_open, dyn Fn());
	set!(set_onclose, on_close, dyn Fn(CloseEvent));
	set!(set_onerror, on_error, dyn Fn(ErrorEvent));
	set!(set_onerror, on_message, dyn Fn(MessageEvent));

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
