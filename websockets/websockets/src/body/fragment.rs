use super::Result;

pub fn parse_fragment(frag: &str, flags: u8) -> Result {
	use protocol::payloads::post_body::Node;

	let mut out = vec![];
	let mut text = String::new();
	for (i, word) in frag.split(' ').enumerate() {
		if i != 0 {
			text.push(' ');
		}

		// Split off leading and trailing punctuation, if any
		let (lead, word, trail) = split_punctuation(&word);
		if lead != 0 as char {
			text.push(lead);
		}

		let matched: Option<Node> = match word.chars().next() {
			None => {
				if trail != 0 as char {
					text.push(trail);
				}
				continue;
			}

			// Hash commands
			Some('#') => {
				if flags & (super::QUOTED | super::OPEN) != 0 {
					None
				} else {
					use super::{
						commands::*, AUTOBAHN_PREFIX, COUNTDOWN_PREFIX,
					};
					use protocol::payloads::post_body::PendingNode::*;

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
			Some('>') => super::links::parse_link(word, flags)?,

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
							for _ in 0..=start {
								text.push('>');
							}
						}
						n
					})
					.flatten()
			}
		};
		match matched {
			None => {
				text += word;
			}
			Some(n) => {
				if !text.is_empty() {
					out.push(Node::Text(std::mem::take(&mut text)));
				}
				out.push(n);
			}
		}
		if trail != 0 as char {
			text.push(trail);
		}
	}
	if !text.is_empty() {
		out.push(Node::Text(text))
	}

	// Pack vector as siblings
	Ok(out.into_iter().rev().fold(Node::Empty, |right, left| {
		if matches!(right, Node::Empty) {
			left
		} else {
			Node::Siblings([left.into(), right.into()])
		}
	}))
}

/// Split off one byte of leading and trailing punctuation, if any, and returns
/// the 3 split parts. If there is no edge punctuation, the respective byte is
/// zero.
fn split_punctuation(word: &str) -> (char, &str, char) {
	fn is_punctuation(b: char) -> bool {
		match b {
			'!' | '"' | '\'' | '(' | ')' | ',' | '-' | '.' | ':' | ';'
			| '?' | '[' | ']' => true,
			_ => false,
		}
	}

	let mut out = (0 as char, word, 0 as char);

	// Split off leading
	if out.1.len() < 2 {
		return out;
	}
	match out.1.chars().next() {
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
	match out.1.chars().rev().next() {
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
