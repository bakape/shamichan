use super::Result;
use protocol::payloads::post_body::Node;

// Implements a function that wraps matched content in a tag
macro_rules! impl_wrapper {
	($vis:vis, $name:ident, $delimiter:expr, $tag:ident, $inner:expr) => {
		$vis fn $name(frag: &str, flags: u8) -> Result {
			super::util::split_and_parse(
				frag,
				flags,
				$delimiter,
				|frag, flags| -> Result {
					Ok(Node::$tag($inner(frag, flags)?.into()))
				},
				$inner,
			)
		}
	};
}

impl_wrapper! {
	pub,
	parse_spoilers,
	"**",
	Spoiler,
	parse_bolds
}

impl_wrapper! {
	,
	parse_bolds,
	"@@",
	Bold,
	parse_italics
}

impl_wrapper! {
	,
	parse_italics,
	"~~",
	Italic,
	parse_quoted
}

fn parse_quoted(frag: &str, flags: u8) -> Result {
	use super::{
		fragment::parse_fragment, util::parse_siblings, AT_LINE_START, QUOTED,
	};

	fn wrap_quoted(line: &str, flags: u8) -> Result {
		Ok(match line.chars().next() {
			None => Node::Empty,
			Some('>') => {
				Node::Quoted(parse_fragment(line, flags | QUOTED)?.into())
			}
			Some(_) => parse_fragment(line, flags)?,
		})
	}

	let pos = frag.find('\n');
	if flags & AT_LINE_START != 0 {
		match pos {
			Some(pos) => parse_siblings(
				|| wrap_quoted(&frag[..pos], flags & !AT_LINE_START),
				|| parse_quoted(&frag[pos + 1..], flags),
			),
			None => wrap_quoted(frag, flags & !AT_LINE_START),
		}
	} else {
		match pos {
			Some(pos) => parse_siblings(
				|| parse_fragment(&frag[..pos], flags),
				|| parse_quoted(&frag[pos + 1..], flags | AT_LINE_START),
			),
			None => parse_fragment(frag, flags),
		}
	}
}
