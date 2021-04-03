use super::{fragment::parse_fragment, QUOTED};
use common::payloads::post_body::Node;

/// Split by delimiter and run parsing function on the matched and unmatched
/// segments
///
/// Always inlined, because it is only used in very small function that only
/// call split_and_parse().
#[inline(always)]
pub fn split_and_parse(
	dst: &mut Node,
	frag: &str,
	flags: u8,
	delimiter: &str,
	matched: impl FnOnce(&mut Node, &str, u8),
	unmatched: impl FnOnce(&mut Node, &str, u8),
) {
	if frag.is_empty() {
		return;
	}

	let with_remainder = |dst: &mut Node, frag: &str| match frag.find(delimiter)
	{
		Some(end) if end == 0 => (),
		Some(end) if end < frag.len() => {
			matched(dst, &frag[..end], flags);
			parse_code(dst, &frag[end + delimiter.len()..], flags);
		}
		Some(end) => matched(dst, &frag[..end], flags),
		None => matched(dst, frag, flags),
	};

	match frag.find(delimiter) {
		Some(start) => {
			if start == 0 {
				with_remainder(dst, &frag[delimiter.len()..]);
			} else {
				unmatched(dst, &frag[..start], flags);
				with_remainder(dst, &frag[start + delimiter.len()..]);
			}
		}
		None => unmatched(dst, frag, flags),
	};
}

/// Implements a function that parses and applies text formatting to a text
/// fragment
macro_rules! impl_formatting {
	($(
		$vis:vis fn $name:ident(
			$delimiter:expr => $tag:ident($wrapper:expr) || $inner:expr
		)
	)+) => {
		$(
			$vis fn $name(dst: &mut Node, frag: &str, flags: u8) {
				split_and_parse(
					dst,
					frag,
					flags,
					$delimiter,
					$wrapper,
					$inner,
				);
			}
		)+
	};
	($(
		$vis:vis fn $name:ident($delimiter:expr => $tag:ident || $inner:expr)
	)+) => {
		$(
			impl_formatting! {
				$vis fn $name(
					$delimiter => $tag(
						|mut dst: &mut Node, frag: &str, flags: u8| {
							dst += Node::$tag(collect_node(
								frag,
								flags,
								$inner,
							));
						}
					)
					|| $inner
				)
			}
		)+
	};
}

// TODO: don't remove line start flag after formatting
impl_formatting! {
	fn parse_code("``" => Code(highlight_code) || parse_spoilers)
}
impl_formatting! {
	fn parse_spoilers("**" => Spoiler || parse_bolds)
	fn parse_bolds("@@" => Bold || parse_italics)
	fn parse_italics("~~" => Italic || parse_fragment)
}

/// Collect output of inner into a fresh node
#[inline]
fn collect_node(
	frag: &str,
	flags: u8,
	inner: impl FnOnce(&mut Node, &str, u8),
) -> Box<Node> {
	let mut n = Box::new(Node::Empty);
	inner(&mut n, frag, flags);
	n
}

/// Highlight programming code
fn highlight_code(mut dst: &mut Node, mut frag: &str, _: u8) {
	use syntect::{
		html::{ClassStyle, ClassedHTMLGenerator},
		parsing::SyntaxSet,
	};

	lazy_static::lazy_static! {
		static ref SYNTAX_SET: SyntaxSet = SyntaxSet::load_defaults_newlines();
	}

	let mut gen = ClassedHTMLGenerator::new_with_class_style(
		frag.find(|b: char| !b.is_alphabetic())
			.map(|pos| {
				let s = SYNTAX_SET.find_syntax_by_token(&frag[..pos]);
				if s.is_some() {
					frag = &frag[pos + 1..];
				}
				s
			})
			.flatten()
			.or_else(|| SYNTAX_SET.find_syntax_by_first_line(&frag))
			.unwrap_or_else(|| SYNTAX_SET.find_syntax_plain_text()),
		&SYNTAX_SET,
		ClassStyle::SpacedPrefixed { prefix: "syntex-" },
	);

	let mut scratch = String::new();
	for mut line in frag.lines() {
		// Must ensure line ends with newline for syntex compatibility
		match line.bytes().last() {
			Some(b'\n') => (),
			_ => {
				scratch.clear();
				scratch += line;
				scratch.push('\n');
				line = &scratch;
			}
		};
		gen.parse_html_for_line_which_includes_newline(&line)
	}
	let mut html = gen.finalize();

	// If original fragment did not contain any newlines, strip the introduced
	// newline from the formatted html.
	if !frag.bytes().any(|b| b == b'\n') {
		html = std::mem::take(&mut html)
			.chars()
			.filter(|ch| ch != &'\n')
			.collect();
	}

	dst += Node::Code(html);
}

/// Top level parsing function. Parses line by line and detects quotes.
/// Must be called at line start
pub fn parse_quoted(dst: &mut Node, frag: &str, flags: u8) {
	// Close an exiting quotation level and commit any uncommitted text down the
	// parser pipeline
	let close_level = |mut dst: &mut Node,
	                   level: usize,
	                   start: usize,
	                   i: usize,
	                   frag: &str| {
		if start != i {
			if level == 0 {
				// Open quotation block and commit previous unquoted text
				parse_code(dst, &frag[start..i], flags);
			} else {
				dst += Node::Quoted(collect_node(
					// Close quotation block and possibly open quotation block
					// with a different level
					&frag[start..i],
					flags | QUOTED,
					parse_code,
				));
			}
		}
	};

	// Find segments of unquoted text and quoted text of the same quotation
	// level
	let mut start = 0;
	let mut i = 0;
	let mut quote_level = 0;
	while i < frag.len() {
		let line_level = if frag.as_bytes()[i] == b'>' {
			// Account for links right after the quote, when detecting quotation
			// level
			super::links::detect_link(
				&frag[i..frag[i..]
					.bytes()
					.position(|b| b == b'\n' || b == b' ')
					.map(|pos| pos + i)
					.unwrap_or(frag.len())],
			)
			.unwrap_or_else(|| {
				frag[i..].bytes().take_while(|b| b == &b'>').count()
			})
		} else {
			0
		};

		if line_level != quote_level {
			close_level(dst, quote_level, start, i, frag);
			quote_level = line_level;
			start = i;
		}

		i = frag[i..]
			.bytes()
			.position(|b| b == b'\n')
			.map(|pos| {
				// Advance past the newline
				pos + i + 1
			})
			.unwrap_or(frag.len());
	}
	close_level(dst, quote_level, start, i, frag);
}
