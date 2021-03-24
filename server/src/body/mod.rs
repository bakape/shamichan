mod commands;
mod formatting;
mod fragment;
mod links;
pub mod persist_open;
mod urls;

pub use links::{cache_locations, KnownPostLocation};

use common::payloads::post_body::Node;

// TODO: unit tests
// TODO: newline handling tests

/// Flags post as open
const OPEN: u8 = 1;

/// Flags current fragment as quote
const QUOTED: u8 = 1 << 1;

const COUNTDOWN_PREFIX: &str = "countdown";
const AUTOBAHN_PREFIX: &str = "autobahn";

/// Parse post body into a Node tree. Different behavior for open and closed
/// posts.
///
/// All performed on one thread to maximize thread locality.
/// Yields of work sharing here are doubtable.
//
// TODO: finalization on post closure should be done with a separate async
// traversal function run by the Client
pub fn parse(body: &str, open: bool) -> Node {
	let mut dst = Node::Empty;
	if !body.is_empty() {
		let mut flags = 0;
		if open {
			flags |= OPEN;
		}
		formatting::parse_quoted(&mut dst, &body, flags);
	}
	dst
}

#[cfg(test)]
mod test {
	macro_rules! test_parsing {
		($( $name:ident($in:literal => $out:expr) )+) => {
			$( mod $name {
				#![allow(unused_imports)]
				#![allow(unused)]
				use crate::body::*;
				use common::payloads::post_body::{Node::{self, *}, PendingNode};

				fn text(s: impl Into<String>) -> Node {
					Node::Text(s.into())
				}

				fn quote(inner: Node) -> Node {
					Node::Quoted(inner.into())
				}

				fn spoiler(inner: Node) -> Node {
					Node::Spoiler(inner.into())
				}

				macro_rules! gen_case {
					($fn_name:ident($open:literal)) => {
						#[test]
						fn $fn_name() {
							let mut conf = crate::config::Config::default();
							conf.public = {
								let mut p = common::config::Public::default();
								p.links = vec![(
									"4ch".to_owned(),
									"https://4channel.org".to_owned(),
								)]
									.into_iter()
									.collect();
								p.into()
							};
							crate::config::set(conf);

							links::cache_locations(
								std::iter::once(KnownPostLocation {
									id: 1,
									thread: 1,
									page: 0,
								}),
							);
							links::register_non_existent_post(3);

							let res = parse($in, $open);
							assert!(
								res == $out,
								"got:      {:#?}\nexpected: {:#?}",
								res,
								$out,
							);
						}
					};
				}

				gen_case! { open(true) }
				gen_case! { closed(false) }
			})+
		};
	}

	/// Create a list of child nodes
	macro_rules! children {
		($($ch:expr),*$(,)?) => {
			Node::Children(vec![ $($ch,)* ])
		};
	}

	test_parsing! {
		simple("foo\nbar" => children![
			text("foo"),
			NewLine,
			text("bar"),
		])
		quote(">foo\nbar" => children![
			quote(children![
				text(">foo"),
				NewLine,
			]),
			text("bar"),
		])
		quote_with_multiple_gt(">>foo\nbar" => children![
			quote(children![text(">>foo"), NewLine]),
			text("bar"),
		])
		spoiler("foo**bar** baz" => children![
			text("foo"),
			spoiler(text("bar")),
			text(" baz"),
		])
		multiline_spoiler("**foo\nbar**baz" => children![
			spoiler(children![
				text("foo"),
				NewLine,
				text("bar"),
			]),
			text("baz"),
		])
		unclosed_spoiler_tags("**foo" => spoiler(text("foo")))
		unclosed_multiline_spoiler_tags("**foo\nbar" => spoiler(children![
			text("foo"),
			NewLine,
			text("bar"),
		]))
		spoiler_in_quote(">baz **foo** bar" => quote(children![
			text(">baz "),
			spoiler(text("foo")),
			text(" bar"),
		]))
		spoiler_with_space("**foo **bar" => children![
			spoiler(text("foo ")),
			text("bar"),
		])
		post_link_right_after_quote(">>>1" => quote(children![
			text(">"),
			PostLink {
				id: 1,
				thread: 1,
				page: 0,
			},
		]))
		pending_post_link_right_after_quote(">>>2" => quote(children![
			text(">"),
			Pending(PendingNode::PostLink(2)),
		]))
		invalid_post_link_after_quote(">>>3" => quote(text(">>>3")))
		reference_right_after_quote(">>>>/4ch/" => quote(children![
			text(">"),
			Reference {
				label: "4ch".into(),
				url: "https://4channel.org".into(),
			},
		]))
		post_link_on_unquoted_line(">>1 a" => children![
			PostLink {
				id: 1,
				thread: 1,
				page: 0,
			},
			text(" a"),
		])
		reference_on_unquoted_line(">>>/4ch/ a" => children![
			Reference {
				label: "4ch".into(),
				url: "https://4channel.org".into(),
			},
			text(" a"),
		])
		spoiler_starting_in_line_middle_and_closing_on_the_next(
			"foo **bar\nbaz** woo" => children![
				text("foo "),
				spoiler(children![
					text("bar"),
					NewLine,
					text("baz"),
				]),
				text(" woo"),
			]
		)
		spoiler_starting_in_line_middle_and_never_closing(
			"foo **bar\nbaz woo" => children![
				text("foo "),
				spoiler(children![
					text("bar"),
					NewLine,
					text("baz woo"),
				]),
			]
		)
		spoiler_starting_in_quote_middle_and_closing_on_next(
			">foo **bar\n>baz** woo" => quote(children![
				text(">foo "),
				spoiler(children![
					text("bar"),
					NewLine,
					text(">baz"),
				]),
				text(" woo"),
			])
		)
		spoiler_starting_in_quote_middle_and_never_closing(
			">foo **bar\n>baz woo" =>quote(children![
				text(">foo "),
				spoiler(children![
					text("bar"),
					NewLine,
					text(">baz woo"),
				]),
			])
		)
		spoilers_on_multiple_quotation_levels(
			"**lol\n>foo **bar\n>baz woo\n>>EHHHHHHH" => children![
				spoiler(children![
					text("lol"),
					NewLine,
				]),
				quote(children![
					text(">foo "),
					spoiler(children![
						text("bar"),
						NewLine,
						text(">baz woo"),
						NewLine,
					]),
				]),
				quote(text(">>EHHHHHHH")),
			]
		)
		multiline_bold_tags("foo @@bar\nbaz@@ foo" => children![
			text("foo "),
			Bold(
				children![
					text("bar"),
					NewLine,
					text("baz"),
				]
				.into(),
			),
			text(" foo"),
		])
		multiline_italic_tags("foo ~~bar\nbaz~~ foo" => children![
			text("foo "),
			Italic(
				children![
					text("bar"),
					NewLine,
					text("baz"),
				]
				.into(),
			),
			text(" foo"),
		])
		nested_overlapping_formatting("foo** bar@@b~~a@@zer**h" => children![
			text("foo"),
			spoiler(children![
				text(" bar"),
				Bold(
					children![
						text("b"),
						Italic(text("a").into()),
					]
					.into(),
				),
				text("zer"),
			]),
			text("h"),
		])
		trailing_empty_line("foo\n" => children![text("foo"), NewLine])

			// 		//
	// 		// 	"#flip",
	// 		// 	open: false,
	// 		// 	input: "",
	// 		// 	output: Node::Pending(PendingNode::Flip),
	// 		// },
	// 		//
	// 		// 	"#8ball",
	// 		// 	open: false,
	// 		// 	input: "#8ball",
	// 		// 	output: Node::Pending(PendingNode::EightBall),
	// 		// },
	// 		//
	// 		// 	"edge punctuation",
	// 		// 	open: false,
	// 		// 	input: "(#8ball?",
	// 		// 	output: Node::Children(vec![
	// 		// 		Node::text("("),
	// 		// 		Node::Pending(PendingNode::EightBall),
	// 		// 		Node::text("?"),
	// 		// 	]),
	// 		// },
	// 		// TODO: commands in quote lines
	// 		// TODO: same line code tags
	// 		// TODO: multiline code tags
	// {
	// 	name: "#pyu",
	// 	in:   "#pyu",
	// 	out:  "<strong>#pyu (1)</strong>",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Pyu,
	// 			Pyu:  1,
	// 		},
	// 	},
	// },
	// {
	// 	name: "#pcount",
	// 	in:   "#pcount",
	// 	out:  "<strong>#pcount (2)</strong>",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Pcount,
	// 			Pyu:  2,
	// 		},
	// 	},
	// },
	// {
	// 	name:     "#autobahn",
	// 	in:       "#autobahn",
	// 	out:      "<strong class=\"dead\">#autobahn</strong>",
	// 	commands: []common.Command{{Type: common.Autobahn}},
	// },
	// {
	// 	name: "single roll dice",
	// 	in:   "#d20",
	// 	out:  "<strong>#d20 (21)</strong>",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Dice,
	// 			Dice: []uint16{21},
	// 		},
	// 	},
	// },
	// {
	// 	name: "dubs roll dice",
	// 	in:   "#d20",
	// 	out:  "<strong class=\"dubs_roll\">#d20 (11)</strong>",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Dice,
	// 			Dice: []uint16{11},
	// 		},
	// 	},
	// },
	// {
	// 	name: "max roll dice",
	// 	in:   "#d20",
	// 	out:  "<strong class=\"super_roll\">#d20 (20)</strong>",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Dice,
	// 			Dice: []uint16{20},
	// 		},
	// 	},
	// },
	// {
	// 	name: "multiple roll dice",
	// 	in:   "#2d20",
	// 	out:  "<strong>#2d20 (21 + 33 = 54)</strong>",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Dice,
	// 			Dice: []uint16{21, 33},
	// 		},
	// 	},
	// },
	// {
	// 	name: "too many dice rolls",
	// 	in:   "#11d20",
	// 	out:  "#11d20",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Dice,
	// 			Dice: []uint16{22, 33},
	// 		},
	// 	},
	// },
	// {
	// 	name: "too many dice faces",
	// 	in:   "#2d10001",
	// 	out:  "#2d10001",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Dice,
	// 			Dice: []uint16{22, 33},
	// 		},
	// 	},
	// },
	// {
	// 	name: "no valid commands",
	// 	in:   "#flip",
	// 	out:  "#flip",
	// },
	// {
	// 	name: "too few commands",
	// 	in:   "#flip\n#flip",
	// 	out:  "<strong>#flip (flap)</strong><br>#flip",
	// 	commands: []common.Command{
	// 		{
	// 			Type: common.Flip,
	// 			Flip: true,
	// 		},
	// 	},
	// },
	// {
	// 	name: "no links in post",
	// 	in:   ">>20",
	// 	out:  "<em>>>20</em>",
	// },
	// {
	// 	name:  "1 invalid link",
	// 	in:    ">>20",
	// 	out:   "<em>>>20</em>",
	// 	links: []common.Link{{21, 21, "a"}},
	// },
	// {
	// 	name:  "valid link",
	// 	in:    ">>21",
	// 	out:   `<em><a class="post-link" data-id="21" href="#p21">>>21</a><a class="hash-link" href="#p21"> #</a></em>`,
	// 	op:    20,
	// 	links: []common.Link{{21, 20, "a"}},
	// },
	// {
	// 	name:  "valid link with extra quotes",
	// 	in:    ">>>>21",
	// 	out:   `<em>>><a class="post-link" data-id="21" href="#p21">>>21</a><a class="hash-link" href="#p21"> #</a></em>`,
	// 	op:    20,
	// 	links: []common.Link{{21, 20, "a"}},
	// },
	// {
	// 	name:  "valid cross-thread link",
	// 	in:    ">>21",
	// 	out:   `<em><a class="post-link" data-id="21" href="/c/22#p21">>>21 ➡</a><a class="hash-link" href="/c/22#p21"> #</a></em>`,
	// 	op:    20,
	// 	links: []common.Link{{21, 22, "c"}},
	// },
	// {
	// 	name: "invalid reference",
	// 	in:   ">>>/fufufu/",
	// 	out:  `<em>>>>/fufufu/</em>`,
	// },
	// {
	// 	name: "link reference",
	// 	in:   ">>>/4chan/",
	// 	out:  `<em><a rel="noreferrer" href="http://4chan.org" target="_blank">&gt;&gt;&gt;/4chan/</a></em>`,
	// },
	// {
	// 	name: "board reference",
	// 	in:   ">>>/a/",
	// 	out:  `<em><a rel="noreferrer" href="/a/" target="_blank">&gt;&gt;&gt;/a/</a></em>`,
	// },
	// {
	// 	name: "reference with extra quotes",
	// 	in:   ">>>>>/a/",
	// 	out:  `<em>>><a rel="noreferrer" href="/a/" target="_blank">&gt;&gt;&gt;/a/</a></em>`,
	// },
	// {
	// 	name: "HTTP URL",
	// 	in:   "http://4chan.org",
	// 	out:  `<a rel="noreferrer" href="http://4chan.org" target="_blank">http://4chan.org</a>`,
	// },
	// {
	// 	name: "HTTPS URL",
	// 	in:   "https://4chan.org",
	// 	out:  `<a rel="noreferrer" href="https://4chan.org" target="_blank">https://4chan.org</a>`,
	// },
	// {
	// 	name: "magnet URL",
	// 	in:   "magnet:?xt=urn:btih:c12fe1",
	// 	out:  `<a rel="noreferrer" href="magnet:?xt=urn:btih:c12fe1">magnet:?xt=urn:btih:c12fe1</a>`,
	// },
	// {
	// 	name: "escape generic text",
	// 	in:   "<>&",
	// 	out:  "&lt;&gt;&amp;",
	// },
	// {
	// 	name: "youtu.be embed",
	// 	in:   "https://youtu.be/z0f4Wgi94eo",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"0\" href=\"https://youtu.be/z0f4Wgi94eo\">[YouTube] ???</a></em>",
	// },
	// {
	// 	name: "youtube embed",
	// 	in:   "https://www.youtube.com/embed/z0f4Wgi94eo",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"0\" href=\"https://www.youtube.com/embed/z0f4Wgi94eo\">[YouTube] ???</a></em>",
	// },
	// {
	// 	name: "youtube embed",
	// 	in:   "https://www.youtube.com/watch?v=z0f4Wgi94eo",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"0\" href=\"https://www.youtube.com/watch?v=z0f4Wgi94eo\">[YouTube] ???</a></em>",
	// },
	// {
	// 	name: "soundcloud embed",
	// 	in:   "https://soundcloud.com/cd_oblongar",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"1\" href=\"https://soundcloud.com/cd_oblongar\">[SoundCloud] ???</a></em>",
	// },
	// {
	// 	name: "vimeo embed",
	// 	in:   "https://vimeo.com/174312494",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"2\" href=\"https://vimeo.com/174312494\">[Vimeo] ???</a></em>",
	// },
	// {
	// 	name: "bitchute embed",
	// 	in:   "https://www.bitchute.com/embed/z0f4Wgi94eo",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"4\" href=\"https://www.bitchute.com/embed/z0f4Wgi94eo\">[BitChute] ???</a></em>",
	// },
	// {
	// 	name: "bitchute embed",
	// 	in:   "https://www.bitchute.com/video/z0f4Wgi94eo",
	// 	out:  "<em><a rel=\"noreferrer\" class=\"embed\" target=\"_blank\" data-type=\"4\" href=\"https://www.bitchute.com/video/z0f4Wgi94eo\">[BitChute] ???</a></em>",
	// },
	}
}
