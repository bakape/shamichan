use super::Result;

// Programming code tags are top level and can override everything else
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
	use std::sync::Once;
	use syntect::{
		highlighting::{Theme, ThemeSet},
		parsing::SyntaxSet,
	};

	static ONCE: Once = Once::new();
	static mut SYNTAX_SET: Option<SyntaxSet> = None;
	static mut THEME: Option<Theme> = None;
	ONCE.call_once(|| unsafe {
		SYNTAX_SET = SyntaxSet::load_defaults_nonewlines().into();
		THEME = ThemeSet::load_defaults()
			.themes
			.remove("base16-eighties.dark");
	});

	let ss = unsafe { SYNTAX_SET.as_ref().unwrap() };
	Ok(protocol::payloads::post_body::Node::Code(
		match frag
			.find(' ')
			.map(|pos| ss.find_syntax_by_token(&frag[..pos]))
			.flatten()
		{
			Some(syntax) => {
				let mut gen =
					syntect::html::ClassedHTMLGenerator::new(syntax, ss);
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
