/// Multithreaded recursive decent post text body parser and differ
// TODO: port parser tests from v6
// TODO: Write differ tests
mod code;
mod commands;
mod formatting;
mod fragment;
mod links;
pub mod persist_open;
mod urls;
mod util;

pub use links::cache_location;

use common::payloads::post_body::Node;

// TODO: unit tests
// TODO: newline handling tests

/// Flags post as open
const OPEN: u8 = 1;

/// Flags current fragment as quoted
const QUOTED: u8 = 1 << 1;

/// Currently parser at the start of the body
const AT_LINE_START: u8 = 1 << 2;

const COUNTDOWN_PREFIX: &str = "countdown";
const AUTOBAHN_PREFIX: &str = "autobahn";

/// Parsing result shorthand
pub type Result<T = Node> = std::result::Result<T, String>;

/// Parse post body into a Node tree. Different behavior for open and closed
/// posts.
///
/// All performed on one thread to maximize thread locality.
/// Yields of work sharing here are doubtable.
//
// TODO: finalization on post closure should be done with a separate async
// traversal function run by the Client
pub fn parse(body: &str, open: bool) -> Result {
	if body.len() > 2000 {
		// Best have an extra guard just in case to protect from parsing large
		// strings
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
