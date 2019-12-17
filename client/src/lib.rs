mod connection;
mod fsm;
mod state;
mod util;

use brunhild::*;
use wasm_bindgen::prelude::*;

#[wasm_bindgen(start)]
pub fn main_js() -> Result<(), JsValue> {
	console_error_panic_hook::set_once();

	Node::text(&TextOptions {
		text: "TESTO",
		..Default::default()
	})
	.append_to(util::body().into())?;

	state::with(|s| {
		connection::init(s);
	});

	Ok(())
}
