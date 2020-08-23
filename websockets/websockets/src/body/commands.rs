use super::{AUTOBAHN_PREFIX, COUNTDOWN_PREFIX};
use protocol::payloads::post_body::{Node, PendingNode};

/// Read command with an optional argument enclosed in ()
#[inline(always)]
fn parse_argument_command<T>(
	word: &str,
	prefix: &str,
	default: T,
	gen_command: impl FnOnce(T) -> PendingNode,
) -> Option<Node>
where
	T: std::str::FromStr,
{
	let wrap = |val: T| Node::Pending(gen_command(val));
	let arg = &word[prefix.len()..];

	if arg.is_empty() {
		Some(wrap(default))
	} else if arg.starts_with('(') && arg.ends_with(')') {
		arg[1..arg.len() - 1].parse().ok().map(wrap)
	} else {
		None
	}
}

/// Parse countdown timer hash command
pub fn parse_countdown(word: &str) -> Option<Node> {
	parse_argument_command(word, COUNTDOWN_PREFIX, 10, |secs| {
		PendingNode::Countdown(secs)
	})
}

/// Parse autobahn timer hash command
pub fn parse_autobahn(word: &str) -> Option<Node> {
	parse_argument_command(word, AUTOBAHN_PREFIX, 2, |hours| {
		PendingNode::Autobahn(hours)
	})
}

/// Parse dice roll hash command
pub fn parse_dice(word: &str) -> Option<Node> {
	let d_pos = match word.find('d') {
		Some(i) => i,
		None => return None,
	};
	let sign_pos = word[d_pos + 1..].find(|c| c == '+' || c == '-');

	#[rustfmt::skip]
	macro_rules! parse {
		($s:expr) => {
			match $s.parse().ok() {
				Some(i) => i,
				None => return None,
			}
		};
	}

	Some(Node::Pending(PendingNode::Dice {
		rolls: if d_pos == 0 {
			1
		} else {
			let r = parse!(word[..d_pos]);
			if r > 10 {
				return None;
			}
			r
		},
		faces: parse!(word[d_pos + 1..sign_pos.unwrap_or(word.len())]),
		offset: match sign_pos {
			Some(i) => parse!(word[i..]),
			None => 0,
		},
	}))
}
