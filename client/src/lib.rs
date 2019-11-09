mod util;

use brunhild::*;
use wasm_bindgen::prelude::*;

#[wasm_bindgen(start)]
pub fn main_js() -> Result<(), JsValue> {
	// This provides better error messages in debug mode.
	// It's disabled in release mode so it doesn't bloat up the file size.
	#[cfg(debug_assertions)]
	console_error_panic_hook::set_once();

	Node::text(&TextOptions {
		text: "TESTO",
		..Default::default()
	})
	.append_to(util::body().into())?;

	Ok(())
}
