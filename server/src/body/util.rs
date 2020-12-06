use super::{Result, AT_LINE_START};
use common::payloads::post_body::Node;

/// Parse a split sibling pair (if resources allow)
pub fn parse_siblings(
	left: impl FnOnce() -> Result + Send,
	right: impl FnOnce() -> Result + Send,
) -> Result {
	let (left_res, right_res) = rayon::join(left, right);

	// Avoid  and extra Node and boxing, if one is empty
	let left_n = left_res?;
	let right_n = right_res?;
	Ok(match (&left_n, &right_n) {
		(_, Node::Empty) => left_n,
		(Node::Empty, _) => right_n,
		_ => Node::Siblings([left_n.into(), right_n.into()]),
	})
}

/// Split by delimiter and run parsing function on the matched and unmatched
/// segments
///
/// Always inlined, because it is only used in very small function that only
/// call this split_and_parse().
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
