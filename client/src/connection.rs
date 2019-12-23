use super::cache_cb;
use super::fsm::FSM;
use super::state;
use super::state::State;
use super::util;
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

fn render_status(s: ConnState) {
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
	}))
}

fn close_socket(s: &mut State) {
	if let Some(s) = &s.socket {
		s.close().expect("failed to closed socket");
	}
	s.socket = None;
}

fn alert_error(reason: &str) {
	if reason != "" {
		util::window()
			.alert_with_message(&format!("connection closed: {}", reason,))
			.expect("alert failed");
	}
}

// Run closure with application and connection state as arguments
fn with_state<F>(f: F)
where
	F: Fn(&mut State, &mut FSM<ConnState, ConnEvent>),
{
	state::with(|s| with(|c| f(s, c)));
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
	.unwrap();

	macro_rules! set {
		($prop:ident, $type:ty, $fn:expr) => {
			socket.$prop(Some(cache_cb!($type, $fn)));
		};
	}

	set!(set_onopen, dyn Fn(), || {
		with_state(|s, c| c.feed(s, ConnEvent::Open))
	});
	set!(set_onclose, dyn Fn(CloseEvent), |e: CloseEvent| {
		with_state(|s, c| {
			if e.code() != 1000 {
				alert_error(&e.reason());
			}
			c.feed(s, ConnEvent::Close);
		})
	});
	set!(set_onerror, dyn Fn(ErrorEvent), |e: ErrorEvent| {
		with_state(|s, c| {
			alert_error(&e.message());
			c.feed(s, ConnEvent::Error);
		})
	});
	set!(set_onerror, dyn Fn(MessageEvent), |e: MessageEvent| {
		// TODO
		web_sys::console::log_1(e.unchecked_ref());
	});

	s.socket = Some(socket);
}

// Reset module state
fn reset(s: &mut State) {
	close_socket(s);
	reset_reconn_timer(s);
}

fn reset_reconn_timer(s: &mut State) {
	if s.reconn_timer != 0 {
		util::window().clear_timeout_with_handle(s.reconn_timer);
		s.reconn_timer = 0;
	}
}

fn reset_reconn_attempts(s: &mut State) {
	reset_reconn_timer(s);
	s.reconn_attempts = 0;
}

// Initiate websocket connection to server
pub fn init(state: &mut State) {
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
			&[ConnEvent::Open],
			&|s, _, _| {
				// TODO: Synchronize

				reset_reconn_attempts(s);
				ConnState::Syncing
			},
		);

		c.set_transitions(
			&[ConnState::Syncing],
			&[ConnEvent::Sync],
			&|_, _, _| ConnState::Synced,
		);

		c.set_any_state_transitions(&[ConnEvent::Close], &|s, state, _| {
			reset(s);

			s.reconn_attempts += 1;
			util::window()
				.set_timeout_with_callback_and_timeout_and_arguments_0(
					cache_cb!(dyn Fn(), || {
						with_state(|s, c| c.feed(s, ConnEvent::Retry))
					}),
					// Maxes out at ~1min
					(500f32
						* 1.5f32.powi(std::cmp::min(s.reconn_attempts / 2, 12)))
						as i32,
				)
				.unwrap();

			if state == ConnState::Desynced {
				ConnState::Desynced
			} else {
				ConnState::Dropped
			}
		});

		c.set_transitions(
			&[ConnState::Dropped],
			&[ConnEvent::Retry],
			&|s, _, _| {
				if util::window().navigator().on_line() {
					connect(s);
					ConnState::Reconnecting
				} else {
					ConnState::Dropped
				}
			},
		);

		c.set_any_state_transitions(&[ConnEvent::Error], &|s, _, _| {
			reset(s);
			ConnState::Desynced
		});

		c.feed(state, ConnEvent::Start);

		// Work around browser slowing down/suspending tabs and keep the FSM up
		// to date with the actual tab status
		util::add_listener(
			util::document(),
			"visibilitychange",
			super::event_handler!(|_| {
				with_state(|s, c| {
					if util::document().hidden()
						|| !util::window().navigator().on_line()
					{
						return;
					}
					match c.state() {
						// Ensure still connected, in case the computer went
						// to sleep or hibernate or the mobile browser tab
						// was suspended
						ConnState::Synced => {
							// TODO: Send ping to server
						}
						ConnState::Desynced => return,
						_ => c.feed(s, ConnEvent::Retry),
					};
				});
			}),
		);

		util::add_listener(
			util::window(),
			"online",
			super::event_handler!(|_| {
				with_state(|s, c| {
					reset_reconn_attempts(s);
					c.feed(s, ConnEvent::Retry);
				})
			}),
		);
		util::add_listener(
			util::window(),
			"offline",
			super::event_handler!(|_| {
				with_state(|s, c| c.feed(s, ConnEvent::Close))
			}),
		);
	});
}
