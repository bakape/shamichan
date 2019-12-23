use super::util;
use std::mem::{self, MaybeUninit};
use std::str;

// Global state singleton
#[derive(Default)]
pub struct State {
	// Reconnection attempts
	pub reconn_attempts: i32,

	// Reconnection timer ID
	pub reconn_timer: i32,

	// Connection to server
	pub socket: Option<web_sys::WebSocket>,

	// Authentication key
	pub auth_key: protocol::AuthKey,
}

super::gen_global!(State);

impl State {
	// Read saved or generate a new authentication key
	pub fn load_auth_key(&mut self) -> util::JSResult {
		let ls = util::local_storage();
		const KEY: &str = "auth_key";
		match ls.get_item(KEY)? {
			Some(v) => {
				base64::decode_config_slice(
					&v,
					base64::STANDARD,
					self.auth_key.as_mut_slice(),
				)
				.map_err(|e| e.to_string())?;
			}
			None => {
				util::window().crypto()?.get_random_values_with_u8_array(
					self.auth_key.as_mut_slice(),
				)?;
				let mut buf: [u8; 88] =
					unsafe { MaybeUninit::uninit().assume_init() };
				base64::encode_config_slice(
					self.auth_key.as_mut_slice(),
					base64::STANDARD,
					&mut buf,
				);
				ls.set_item(KEY, unsafe { str::from_utf8_unchecked(&buf) })?;
			}
		};

		Ok(())
	}
}
