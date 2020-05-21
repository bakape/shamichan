// Multithreaded recursive decent post text body parser and differ

// TODO: port parser tests from v6
// TODO: Write differ tests

mod code;
mod commands;
mod diff;
mod formatting;
mod fragment;
mod links;
mod persist;
mod urls;
mod util;

pub use diff::diff;
pub use links::cache_location;
pub use persist::persist_open_body;

use protocol::payloads::post_body::Node;

// Flags post as open
const OPEN: u8 = 1;

// Flags current fragment as quoted
const QUOTED: u8 = 1 << 1;

// Currently parser at the start of the body
const AT_LINE_START: u8 = 1 << 2;

const COUNTDOWN_PREFIX: &str = "countdown";
const AUTOBAHN_PREFIX: &str = "autobahn";

// Parsing result shorthand
pub type Result<T = Node> = std::result::Result<T, String>;

// Parse post body into a Node tree. Different behavior for open and closed
// posts.
pub fn parse(body: &str, open: bool) -> Result {
	if body.len() > 2000 {
		Err("post body too long".into())
	} else if body.is_empty() {
		Ok(Node::Empty)
	} else {
		let mut flags = AT_LINE_START;
		if open {
			flags |= OPEN;
		}
		code::parse_code(&body, flags)
	}
}
