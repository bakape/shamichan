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
	use syntect::parsing::SyntaxSet;

	lazy_static! {
		static ref SYNTAX_SET: SyntaxSet =
			SyntaxSet::load_defaults_nonewlines();
	}

	Ok(protocol::payloads::post_body::Node::Code(
		match frag
			.find(' ')
			.map(|pos| SYNTAX_SET.find_syntax_by_token(&frag[..pos]))
			.flatten()
		{
			Some(syntax) => {
				use syntect::html::{ClassStyle, ClassedHTMLGenerator};

				let mut gen = ClassedHTMLGenerator::new_with_class_style(
					syntax,
					&SYNTAX_SET,
					ClassStyle::Spaced,
				);
				for line in frag.lines() {
					gen.parse_html_for_line(&line);
				}
				gen.finalize()
			}
			None => {
				// String will be 24 bytes longer, if no escaping occurs
				let mut w = String::with_capacity(frag.len() + 24);
				w += "<pre class=\"code\">";
				htmlescape::encode_minimal_w(frag, unsafe { w.as_mut_vec() })
					.map_err(|e| e.to_string())?;
				w += "</pre>";
				w
			}
		},
	))
}
