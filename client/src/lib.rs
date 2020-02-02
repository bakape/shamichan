mod connection;
mod fsm;
mod lang;
mod page;
mod state;
mod util;

#[macro_use]
extern crate protocol;

use wasm_bindgen::prelude::*;

#[wasm_bindgen(start)]
pub async fn main_js() -> util::Result {
	console_error_panic_hook::set_once();

	async fn run(s: &mut state::State) -> util::Result {
		s.thread = util::window().location().hash()?.parse().unwrap_or(0);

		s.load_auth_key()?;
		lang::load_language_pack().await?;

		connection::init(s)?;

		// Mount the container views
		s.views
			.aside_top
			.mount_as(&util::get_el("aside-container-top"))?;
		s.views
			.aside_bottom
			.mount_as(&util::get_el("aside-container-bottom"))?;

		page::render(s)
	}

	state::with(run).await?;

	Ok(())
}
