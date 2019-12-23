mod connection;
mod fsm;
mod lang;
mod state;
mod util;

use wasm_bindgen::prelude::*;

#[wasm_bindgen(start)]
pub async fn main_js() -> Result<(), JsValue> {
	console_error_panic_hook::set_once();

	async fn run(s: &mut state::State) -> Result<(), JsValue> {
		lang::load_language_pack().await?;

		connection::init(s);
		Ok(())
	}

	state::with(run).await?;

	Ok(())
}
