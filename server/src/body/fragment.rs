use common::payloads::post_body::Node;

/// Parse a text fragment not containing any formatting tags
pub fn parse_fragment(dst: &mut Node, frag: &str, flags: u8) {
	if frag.is_empty() {
		return;
	}

	for (i, frag) in frag.split("\n").enumerate() {
		if i != 0 {
			*dst += Node::Newline;
		}
		parse_line_fragment(dst, frag, flags);
	}
}

/// Parse fragment of text not containing newlines
fn parse_line_fragment(dst: &mut Node, frag: &str, flags: u8) {
	if frag.is_empty() {
		return;
	}

	for (i, word_orig) in frag.split(' ').enumerate() {
		if i != 0 {
			*dst += ' ';
		}

		// Split off leading and trailing punctuation, if any
		let (lead, word, mut trail) = split_punctuation(&word_orig);
		if lead != 0 {
			*dst += lead;
		}

		if !match word.bytes().next() {
			None => false,
			Some(b'#') if flags & super::QUOTED == 0 => {
				parse_command(dst, word, &word_orig, lead, &mut trail)
			}
			Some(b'>') => super::links::parse_link(dst, word),
			_ => false,
		} {
			parse_word(dst, word, flags);
		}

		if trail != 0 {
			*dst += trail;
		}
	}
}

/// Parse hash commands. Moved to separate function to not bloat the hot
/// parse_line_fragment loop.
///
/// Returns, if a valid command has been parsed and written to dst.
#[cold]
fn parse_command(
	dst: &mut Node,
	word: &str,
	word_with_punctuation: &str,
	leading_punctuation: u8,
	trailing_punctuation: &mut u8,
) -> bool {
	use super::{commands::*, AUTOBAHN_PREFIX, COUNTDOWN_PREFIX};
	use common::payloads::post_body::PendingNode::*;

	/// Generate a command node pending finalization
	macro_rules! push_pending {
		($comm:ident) => {{
			*dst += Node::Pending($comm);
			true
		}};
	}

	let comm = &word[1..];
	match comm {
		"flip" => push_pending!(Flip),
		"8ball" => push_pending!(EightBall),
		"pyu" => push_pending!(Pyu),
		"pcount" => push_pending!(PCount),
		_ => {
			// Revert trailing `)` removal on match
			let mut src = comm;
			if *trailing_punctuation == b')' {
				// Keep `#` stripped
				src = &word_with_punctuation
					[if leading_punctuation != 0 { 2 } else { 1 }..];
			}

			if comm.starts_with(COUNTDOWN_PREFIX) {
				parse_countdown(src)
			} else if comm.starts_with(AUTOBAHN_PREFIX) {
				parse_autobahn(src)
			} else {
				parse_dice(src)
			}
			.map(|n| {
				if *trailing_punctuation == b')' {
					*trailing_punctuation = 0;
				}
				*dst += n;
				true
			})
			.unwrap_or(false)
		}
	}
}

/// Fallback word parser
fn parse_word(dst: &mut Node, word: &str, flags: u8) {
	match word
		.chars()
		.position(|c| c != '>')
		.map(|leading_gt| {
			// Strip any leading '>'
			let word = &word[leading_gt..];

			let n = if word.starts_with("http") {
				super::urls::parse_http_url(word, flags)
			} else if ["magnet:?", "ftp", "bitcoin"]
				.iter()
				.any(|pre| word.starts_with(pre))
			{
				url::Url::parse(word).ok().map(|_| Node::URL(word.into()))
			} else {
				None
			};
			if leading_gt != 0 && n.is_some() {
				*dst += ">".repeat(leading_gt);
			}
			n
		})
		.flatten()
	{
		Some(n) => *dst += n,
		_ => *dst += word,
	};
}

/// Split off one byte of leading and trailing punctuation, if any, and returns
/// the 3 split parts. If there is no edge punctuation, the respective byte is
/// zero.
fn split_punctuation(word: &str) -> (u8, &str, u8) {
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
