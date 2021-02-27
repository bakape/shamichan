use super::{Result, AT_LINE_START};
use common::payloads::post_body::Node;

/// Parse a split sibling pair.
#[inline]
pub fn parse_siblings(
	left: impl FnOnce() -> Result,
	right: impl FnOnce() -> Result,
) -> Result {
	Ok(match (left()?, right()?) {
		// Avoid  and extra Node and boxing, if one is empty
		(l @ _, Node::Empty) => l,
		(Node::Empty, r @ _) => r,

		(l @ _, r @ _) => Node::Siblings([l.into(), r.into()]),
	})
}

/// Split by delimiter and run parsing function on the matched and unmatched
/// segments
///
/// Always inlined, because it is only used in very small function that only
/// call split_and_parse().
#[inline(always)]
pub fn split_and_parse(
	frag: &str,
	flags: u8,
	delimiter: &str,
	matched: impl FnOnce(&str, u8) -> Result + Send,
	unmatched: impl FnOnce(&str, u8) -> Result + Send,
) -> Result {
	if let Some(start) = frag.find(delimiter) {
		let with_remainder = |frag: &str, mut flags: u8| {
			flags &= !AT_LINE_START;

			match frag.find(delimiter) {
				Some(end) => {
					if end == 0 {
						Ok(Node::Empty)
					} else if end < frag.len() {
						parse_siblings(
							|| matched(&frag[..end], flags),
							|| {
								super::code::parse_code(
									&frag[end + delimiter.len()..],
									flags,
								)
							},
						)
					} else {
						matched(&frag[..end], flags)
					}
				}
				None => matched(frag, flags),
			}
		};

		if start == 0 {
			with_remainder(&frag[delimiter.len()..], flags)
		} else {
			parse_siblings(
				|| unmatched(&frag[..start], flags),
				|| with_remainder(&frag[start + delimiter.len()..], flags),
			)
		}
	} else {
		unmatched(frag, flags)
	}
}
