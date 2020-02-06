use super::util;
use std::str;

// Global state singleton
#[derive(Default)]
pub struct State {
	// Authentication key
	pub auth_key: protocol::AuthKey,

	// Currently subscribed to thread. 0 = global thread index
	pub thread: u64,
}

super::gen_global!(pub, State);

impl State {
	// Read saved or generate a new authentication key
	pub fn load_auth_key(&mut self) -> util::Result {
		let ls = util::local_storage();
		const KEY: &str = "auth_key";
		match ls.get_item(KEY)? {
			Some(v) => {
				base64::decode_config_slice(
					&v,
					base64::STANDARD,
					self.auth_key.as_mut(),
				)?;
			}
			None => {
				util::window()
					.crypto()?
					.get_random_values_with_u8_array(self.auth_key.as_mut())?;
				let mut buf: [u8; 88] =
					unsafe { std::mem::MaybeUninit::uninit().assume_init() };
				base64::encode_config_slice(
					self.auth_key.as_mut(),
					base64::STANDARD,
					&mut buf,
				);
				ls.set_item(KEY, unsafe { str::from_utf8_unchecked(&buf) })?;
			}
		};

		Ok(())
	}
}
