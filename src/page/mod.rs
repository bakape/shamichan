mod board;
mod navigation;
use serde_json;

// Initial render of the page
#[no_mangle]
pub extern "C" fn render_page() {
	// TODO
}

pub fn init() -> serde_json::Result<()> {
	navigation::init()
}
