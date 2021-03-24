use common::payloads::post_body::Node;

/// Parse a text fragment not containing any formatting tags
pub fn parse_fragment(mut dst: &mut Node, frag: &str, flags: u8) {
	if frag.is_empty() {
		return;
	}

	for (i, frag) in frag.split("\n").enumerate() {
		if i != 0 {
			dst += Node::NewLine;
		}
		parse_line_fragment(dst, frag, flags);
	}
}

/// Parse fragment of text not containing newlines
fn parse_line_fragment(mut dst: &mut Node, frag: &str, flags: u8) {
	if frag.is_empty() {
		return;
	}

	for (i, word) in frag.split(' ').enumerate() {
		if i != 0 {
			dst += ' ';
		}

		// Split off leading and trailing punctuation, if any
		let (lead, word, trail) = split_punctuation(&word);
		if lead != 0 {
			dst += lead;
		}

		// Not returning a Node so out.push() can be inlined
		let matched = match word.chars().next() {
			None => None,

			// Hash commands
			Some('#') => {
				if flags & super::QUOTED != 0 {
					None
				} else {
					use super::{
						commands::*, AUTOBAHN_PREFIX, COUNTDOWN_PREFIX,
					};
					use common::payloads::post_body::PendingNode::*;

					/// Generate a command node pending finalization
					macro_rules! gen_pending {
						($comm:tt) => {
							Some(Node::Pending($comm))
						};
					}

					let comm = &word[1..];
					match comm {
						"flip" => gen_pending!(Flip),
						"8ball" => gen_pending!(EightBall),
						"pyu" => gen_pending!(Pyu),
						"pcount" => gen_pending!(PCount),
						_ => {
							if comm.starts_with(COUNTDOWN_PREFIX) {
								parse_countdown(comm)
							} else if comm.starts_with(AUTOBAHN_PREFIX) {
								parse_autobahn(comm)
							} else {
								parse_dice(comm)
							}
						}
					}
				}
			}

			// Post links and configured references
			Some('>') => super::links::parse_link(word),

			_ => {
				word.chars()
					.position(|c| c != '>')
					.map(|start| {
						// Ignore any leading '>'
						let word = &word[start..];

						let n = if word.starts_with("http") {
							super::urls::parse_http_url(word, flags)
						} else if ["magnet:?", "ftp", "bitcoin"]
							.iter()
							.any(|pre| word.starts_with(pre))
						{
							url::Url::parse(word)
								.ok()
								.map(|_| Node::URL(word.into()))
						} else {
							None
						};
						if n.is_some() {
							for _ in 0..start {
								dst += '>';
							}
						}
						n
					})
					.flatten()
			}
		};

		match matched {
			Some(n) => dst += n,
			_ => dst += word,
		};
		if trail != 0 {
			dst += trail;
		}
	}
}

/// Split off one byte of leading and trailing punctuation, if any, and returns
/// the 3 split parts. If there is no edge punctuation, the respective byte is
/// zero.
fn split_punctuation(word: &str) -> (u8, &str, u8) {
	#[inline]
	fn is_punctuation(b: u8) -> bool {
		match b {
			b'!' | b'"' | b'\'' | b'(' | b')' | b',' | b'-' | b'.' | b':'
			| b';' | b'?' | b'[' | b']' => true,
			_ => false,
		}
	}

	let mut out = (0 as u8, word, 0 as u8);

	// Split off leading
	if out.1.len() < 2 {
		return out;
	}
	match out.1.bytes().next() {
		Some(b) => {
			if is_punctuation(b) {
				out.0 = b;
				out.1 = &out.1[1..];
			}
		}
		None => unreachable!(),
	}

	// Split off trailing
	if out.1.len() < 2 {
		return out;
	}
	match out.1.bytes().rev().next() {
		Some(b) => {
			if is_punctuation(b) {
				out.2 = b;
				out.1 = &out.1[..out.1.len() - 1];
			}
		}
		None => unreachable!(),
	}

	out
}
