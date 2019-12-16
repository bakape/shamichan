use web_sys;

// Global state singleton
#[derive(Default)]
pub struct State {
	// Reconnection attempts
	pub reconn_attempts: isize,

	// Connection to server
	pub socket: Option<web_sys::WebSocket>,
}

super::gen_global!(State);
