use common::payloads::post_body::{EmbedProvider, Node};
use regex::Regex;

/// Parse a HTTP or HTTPS URL that also might be an embed
pub fn parse_http_url(word: &str, flags: u8) -> Option<Node> {
	if !url::Url::parse(word).is_ok() {
		return None;
	}

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
		static ref PATTERNS: [(EmbedProvider, Regex); 22] = comp_pat! {
			YouTube => "https?://.*\\.youtube\\.com/watch.*"
			YouTube => "https?://.*\\.youtube\\.com/v/.*"
			YouTube => "https?://youtu\\.be/.*"
			Twitter => "https?://twitter\\.com/.*/status/.*"
			Twitter => "https?://.*\\.twitter\\.com/.*/status/.*"
			Twitter => "https?://twitter\\.com/.*/moments/.*"
			Twitter => "https?://.*\\.twitter\\.com/.*/moments/.*"
			Imgur => "https?://imgur\\.com/(?:[^\\\\/]+/)?[0-9a-zA-Z]+$"
			SoundCloud => "https?://soundcloud\\.com/.*"
			SoundCloud => "https?://soundcloud\\.com/.*"
			SoundCloud => "https?://soundcloud\\.app\\.goog\\.gl/.*"
			DropBox => "https?://www\\.(dropbox\\.com/s/.+\\.(?:jpg|png|gif))"
			DropBox => "https?://db\\.tt/[a-zA-Z0-9]+"
			Vimeo => "https?://vimeo\\.com/.*"
			Vimeo => "https?://vimeo\\.com/album/.*/video/.*"
			Vimeo => "https?://vimeo\\.com/channels/.*/.*"
			Vimeo => "https?://vimeo\\.com/groups/.*/videos/.*"
			Vimeo => "https?://vimeo\\.com/ondemand/.*/.*"
			Vimeo => "https?://player\\.vimeo\\.com/video/.*"
			Coub => r#"https?://(?:www\.)?coub\.com/view/.+"#
			BitChute => r#"https?://(?:[^\.]+\.)?(?:bitchute\.com/embed/|bitchute\.com/video/)(:?[a-zA-Z0-9_-]+)"#
			Invidious => r#"https?://(?:www\.)?invidio\.us/watch(:?.*&|\?)v=(:?.+)"#
		};
	}

	if flags & super::OPEN == 0 {
		PATTERNS
			.iter()
			.find(|(_, re)| re.is_match(word))
			.map(|(prov, _)| Node::Embed {
				provider: *prov,
				url: word.into(),
			})
	} else {
		None
	}
	.or_else(|| Some(Node::URL(word.into())))
}
