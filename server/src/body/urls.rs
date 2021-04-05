use common::payloads::post_body::{
	EmbedProvider::{self, *},
	Node,
};
use regex::Regex;

macro_rules! comp_pat {
	($($variant:tt => $regexp:expr)+) => {
		[
			$(
				(EmbedProvider::$variant, Regex::new($regexp).unwrap()),
			)+
		]
	};
}

lazy_static::lazy_static! {
	/// Patterns for matching supported providers, ordered by usage frequency.
	static ref PATTERNS: [(EmbedProvider, Regex); 8] = comp_pat! {
		YouTube => r#"https?://(?:[^\.]+\.)?(?:youtu\.be/|youtube\.com/embed/|youtube\.com/watch\?v=)[a-zA-Z0-9_-]+"#
		Twitter => r#"https?://(?:www|mobile\\.)?twitter\\.com/(?:#!/)?([^/]+)/status(?:es)?/(\\d+)"#
		Imgur => r#"https?://imgur\\.com/(?:[^/]+/)?[0-9a-zA-Z]+$"#
		SoundCloud => r#"https?://soundcloud.com/.*/.*"#
		Vimeo => r#"https?://(?:www\\.)?vimeo\\.com/.+"#
		Coub => r#"https?://(?:www\.)?coub\.com/view/.+"#
		BitChute => r#"https?://(?:[^\.]+\.)?(?:bitchute\.com/embed/|bitchute\.com/video/)([a-zA-Z0-9_-]+)"#
		Invidious => r#"https?://(?:www\.)?invidio\.us/watch(:?.*&|\?)v=(.+)"#
	};
}

/// Parse a HTTP or HTTPS URL that also might be an embed
pub fn parse_http_url(word: &str, flags: u8) -> Option<Node> {
	if !url::Url::parse(word).is_ok() {
		return None;
	}

	if flags & super::OPEN == 0 {
		use common::payloads::post_body::Embed;

		PATTERNS
			.iter()
			.find(|(_, re)| re.is_match(word))
			.map(|(prov, re)| match prov {
				BitChute | Invidious => re
					.captures_iter(word)
					.next()
					.map(|cap| cap.get(1))
					.flatten()
					.map(|cap| {
						Node::Embed(Embed {
							provider: *prov,
							data: cap.as_str().into(),
						})
					}),
				// noembed.com supported providers
				_ => Some(Node::Embed(Embed {
					provider: *prov,
					data: word.into(),
				})),
			})
			.flatten()
	} else {
		None
	}
	.or_else(|| Some(Node::URL(word.into())))
}
