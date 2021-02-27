use super::{util::parse_siblings, Result};
use common::payloads::post_body::Node;

/// Implements a function that wraps matched content in a tag
macro_rules! impl_wrappers {
	($(
		$vis:vis fn $name:ident($delimiter:expr => $tag:ident || $inner:expr)
	)+) => {
		$(
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
		)+
	};
}

impl_wrappers! {
	pub fn parse_spoilers("**" => Spoiler || parse_bolds)
	fn parse_bolds("@@" => Bold || parse_italics)
	fn parse_italics("~~" => Italic || parse_quoted)
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

	#[inline]
	fn append_newline(f: impl Fn() -> Result) -> Result {
		parse_siblings(f, || Ok(Node::NewLine))
	}

	let pos = frag.find('\n');
	if flags & AT_LINE_START != 0 {
		match pos {
			Some(pos) => parse_siblings(
				|| {
					append_newline(|| {
						wrap_quoted(&frag[..pos], flags & !AT_LINE_START)
					})
				},
				|| parse_quoted(&frag[pos + 1..], flags),
			),
			None => wrap_quoted(frag, flags & !AT_LINE_START),
		}
	} else {
		match pos {
			Some(pos) => parse_siblings(
				|| append_newline(|| parse_fragment(&frag[..pos], flags)),
				|| parse_quoted(&frag[pos + 1..], flags | AT_LINE_START),
			),
			None => parse_fragment(frag, flags),
		}
	}
}
