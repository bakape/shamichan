mod connection;
mod fsm;
mod lang;
mod state;
mod util;

use wasm_bindgen::prelude::*;

#[wasm_bindgen(start)]
pub async fn main_js() -> util::Result {
	console_error_panic_hook::set_once();

	async fn run(s: &mut state::State) -> util::Result {
		s.load_auth_key()?;
		lang::load_language_pack().await?;

		connection::init(s)
	}

	state::with(run).await?;

	Ok(())
}
