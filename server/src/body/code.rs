use super::Result;

/// Programming code tags are top level and can override everything else
pub fn parse_code(body: &str, flags: u8) -> Result {
	super::util::split_and_parse(
		&body,
		flags,
		"``",
		highlight_code,
		super::formatting::parse_spoilers,
	)
}

fn highlight_code(frag: &str, _: u8) -> Result {
	use syntect::{
		html::{ClassStyle, ClassedHTMLGenerator},
		parsing::SyntaxSet,
	};

	lazy_static::lazy_static! {
		static ref SYNTAX_SET: SyntaxSet =
			SyntaxSet::load_defaults_nonewlines();
	}

	let mut gen = ClassedHTMLGenerator::new_with_class_style(
		frag.find(' ')
			.map(|pos| SYNTAX_SET.find_syntax_by_token(&frag[..pos]))
			.flatten()
			.or_else(|| SYNTAX_SET.find_syntax_by_first_line(&frag))
			.unwrap_or_else(|| SYNTAX_SET.find_syntax_plain_text()),
		&SYNTAX_SET,
		ClassStyle::Spaced,
	);
	for line in frag.lines() {
		gen.parse_html_for_line_which_includes_newline(&line);
	}
	Ok(common::payloads::post_body::Node::Code(gen.finalize()))
}
