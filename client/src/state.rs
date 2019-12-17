use web_sys;

// Global state singleton
#[derive(Default)]
pub struct State {
	// Reconnection attempts
	pub reconn_attempts: i32,

	// Reconnection timer ID
	pub reconn_timer: i32,

	// Connection to server
	pub socket: Option<web_sys::WebSocket>,
}

super::gen_global!(State);
