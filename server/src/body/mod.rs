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
		($( $name:ident($in:expr => $out:expr) )+) => {
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

				fn code(s: impl Into<String>) -> Node {
					Node::Code(s.into())
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
								"\ngot:      {:#?}\nexpected: {:#?}\n",
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
		edge_punctuation_leading(".#flip" => children![
			text("."),
			Pending(PendingNode::Flip),
		])
		edge_punctuation_trailing("#flip," => children![
			Pending(PendingNode::Flip),
			text(","),
		])
		edge_punctuation_both("(#flip," => children![
			text("("),
			Pending(PendingNode::Flip),
			text(","),
		])
		quoted_command(">#flip" => quote(text(">#flip")))
		flip("#flip" => Pending(PendingNode::Flip))
		eight_ball("#8ball" => Pending(PendingNode::EightBall))
		pyu("#pyu" => Pending(PendingNode::Pyu))
		pcount("#pcount" => Pending(PendingNode::PCount))
		countdown_explicit("#countdown(3)" => Pending(
			PendingNode::Countdown(3)
		))
		countdown_default("#countdown" => Pending(
			PendingNode::Countdown(10)
		))
		failed_command_with_trailing_parenthesis("#countdown_(3)" => text(
			"#countdown_(3)"
		))
		autobahn_explicit("#autobahn(3)" => Pending(
			PendingNode::Autobahn(3)
		))
		autobahn_default("#autobahn" => Pending(
			PendingNode::Autobahn(2)
		))
		code_explicit_language(r#"foo ``python print("bar")`` baz"# => children![
			text("foo "),
			code("<span class=\"syntex-source syntex-python\"><span class=\"syntex-meta syntex-function-call syntex-python\"><span class=\"syntex-meta syntex-qualified-name syntex-python\"><span class=\"syntex-support syntex-function syntex-builtin syntex-python\">print</span></span><span class=\"syntex-punctuation syntex-section syntex-arguments syntex-begin syntex-python\">(</span><span class=\"syntex-meta syntex-function-call syntex-arguments syntex-python\"><span class=\"syntex-meta syntex-string syntex-python\"><span class=\"syntex-string syntex-quoted syntex-double syntex-python\"><span class=\"syntex-punctuation syntex-definition syntex-string syntex-begin syntex-python\">&quot;</span></span></span><span class=\"syntex-meta syntex-string syntex-python\"><span class=\"syntex-string syntex-quoted syntex-double syntex-python\">bar<span class=\"syntex-punctuation syntex-definition syntex-string syntex-end syntex-python\">&quot;</span></span></span></span><span class=\"syntex-punctuation syntex-section syntex-arguments syntex-end syntex-python\">)</span></span></span>"),
			text(" baz"),
		])
		code_guessed_language("``#! /bin/bash\necho \"foo\"``" => code(
			"<span class=\"syntex-source syntex-shell syntex-bash\"><span class=\"syntex-comment syntex-line syntex-number-sign syntex-shell\"><span class=\"syntex-punctuation syntex-definition syntex-comment syntex-begin syntex-shell\">#</span></span><span class=\"syntex-comment syntex-line syntex-number-sign syntex-shell\">! /bin/bash</span><span class=\"syntex-comment syntex-line syntex-number-sign syntex-shell\">\n</span><span class=\"syntex-meta syntex-function-call syntex-shell\"><span class=\"syntex-support syntex-function syntex-echo syntex-shell\">echo</span></span><span class=\"syntex-meta syntex-function-call syntex-arguments syntex-shell\"> <span class=\"syntex-string syntex-quoted syntex-double syntex-shell\"><span class=\"syntex-punctuation syntex-definition syntex-string syntex-begin syntex-shell\">&quot;</span>foo<span class=\"syntex-punctuation syntex-definition syntex-string syntex-end syntex-shell\">&quot;</span></span></span>\n</span>",
		))
		code_cant_guess_language("``foo()``" => code(
			"<span class=\"syntex-text syntex-plain\">foo()</span>",
		))
		code_invalid_explicit_language("``rash foo()``" => code(
			"<span class=\"syntex-text syntex-plain\">rash foo()</span>",
		))
		code_multiline("``bash echo $BAR\neval $BAZ" => code(
			"<span class=\"syntex-source syntex-shell syntex-bash\"><span class=\"syntex-meta syntex-function-call syntex-shell\"><span class=\"syntex-support syntex-function syntex-echo syntex-shell\">echo</span></span><span class=\"syntex-meta syntex-function-call syntex-arguments syntex-shell\"> <span class=\"syntex-meta syntex-group syntex-expansion syntex-parameter syntex-shell\"><span class=\"syntex-punctuation syntex-definition syntex-variable syntex-shell\">$</span><span class=\"syntex-variable syntex-other syntex-readwrite syntex-shell\">BAR</span></span></span>\n<span class=\"syntex-meta syntex-function-call syntex-shell\"><span class=\"syntex-support syntex-function syntex-eval syntex-shell\">eval</span></span><span class=\"syntex-meta syntex-function-call syntex-arguments syntex-shell\"> <span class=\"syntex-meta syntex-group syntex-expansion syntex-parameter syntex-shell\"><span class=\"syntex-punctuation syntex-definition syntex-variable syntex-shell\">$</span><span class=\"syntex-variable syntex-other syntex-readwrite syntex-shell\">BAZ</span></span></span>\n</span>"
		))
		code_multiline_cross_line("foo ``bash echo $BAR\neval $BAZ`` null" => children![
			text("foo "),
			code("<span class=\"syntex-source syntex-shell syntex-bash\"><span class=\"syntex-meta syntex-function-call syntex-shell\"><span class=\"syntex-support syntex-function syntex-echo syntex-shell\">echo</span></span><span class=\"syntex-meta syntex-function-call syntex-arguments syntex-shell\"> <span class=\"syntex-meta syntex-group syntex-expansion syntex-parameter syntex-shell\"><span class=\"syntex-punctuation syntex-definition syntex-variable syntex-shell\">$</span><span class=\"syntex-variable syntex-other syntex-readwrite syntex-shell\">BAR</span></span></span>\n<span class=\"syntex-meta syntex-function-call syntex-shell\"><span class=\"syntex-support syntex-function syntex-eval syntex-shell\">eval</span></span><span class=\"syntex-meta syntex-function-call syntex-arguments syntex-shell\"> <span class=\"syntex-meta syntex-group syntex-expansion syntex-parameter syntex-shell\"><span class=\"syntex-punctuation syntex-definition syntex-variable syntex-shell\">$</span><span class=\"syntex-variable syntex-other syntex-readwrite syntex-shell\">BAZ</span></span></span>\n</span>"),
			text(" null"),
		])
		unknown_reference(">>>/fufufu/" => quote(text(">>>/fufufu/")))
		reference(">>>/4ch/" => Reference{
			label: "4ch".into(),
			url: "https://4channel.org".into(),
		})
		reference_with_extra_gt(">>>>/4ch/" => quote(children![
			text(">"),
			Reference{
				label: "4ch".into(),
				url: "https://4channel.org".into(),
			},
		]))
		reference_with_extra_gt_in_line_middle("f >>>>/4ch/" => children![
			text("f >"),
			Reference{
				label: "4ch".into(),
				url: "https://4channel.org".into(),
			},
		])
		invalid_reference_syntax(">>>/aaa" => quote(text(">>>/aaa")))
		invalid_post_link_syntax(">>3696+" => quote(text(">>3696+")))
		known_existing_post_link(">>1" => PostLink{
			id: 1,
			thread: 1,
			page: 0,
		})
		known_nonexisting_post_link(">>3" => quote(text(">>3")))
		unknown_post_link(">>2" => Pending(PendingNode::PostLink(2)))
		post_link_with_extra_gt(">>>1" => quote(children![
			text(">"),
			PostLink{
				id: 1,
				thread: 1,
				page: 0,
			},
		]))
		post_link_with_extra_ht_in_line_middle("f >>>>1" => children![
			text("f >>"),
			PostLink{
				id: 1,
				thread: 1,
				page: 0,
			},
		])
		empty_quote(">" => quote(text(">")))
		empty_double_quote(">>" => quote(text(">>")))
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
		// TODO: FTP & FTPS URLs
		// {
		// 	name: "magnet URL",
		// 	in:   "magnet:?xt=urn:btih:c12fe1",
		// 	out:  `<a rel="noreferrer" href="magnet:?xt=urn:btih:c12fe1">magnet:?xt=urn:btih:c12fe1</a>`,
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

	mod dice {
		mod valid {
			macro_rules! test_dice_valid {
				($(
					$name:ident($in:literal => {$rolls:literal $faces:literal})
				)+) => {
					$(
						mod $name {
							test_parsing! {
								no_offset(
									$in => 	Pending(PendingNode::Dice{
										offset: 0,
										faces: $faces,
										rolls: $rolls,
									})
								)
								plus_1(
									concat!($in, "+1") => 	Pending(
											PendingNode::Dice{
											offset: 1,
											faces: $faces,
											rolls: $rolls,
										}
									)
								)
								minus_1(
									concat!($in, "-1") => 	Pending(
										PendingNode::Dice{
											offset: -1,
											faces: $faces,
											rolls: $rolls,
										}
									)
								)
							}
						}
					)+
				};
			}

			test_dice_valid! {
				implicit_single_die("#d10" => {1 10})
				explicit_single_die("#1d10" => {1 10})
				explicit_multiple_dice("#2d11" => {2 11})
			}
		}

		mod invalid {
			macro_rules! test_dice_invalid {
				($( $name:ident($in:literal) )+) => {
					test_parsing! {
						$( $name($in => Node::text($in)) )+
					}
				};
			}

			test_dice_invalid! {
				// Dice parser is the final fallback for all unmatched commands
				invalid_command("#ass")
				not_dice("#dagger")

				too_many_dies("#11d6")
				too_many_faces("#d999999999999")
				too_big_offset("#d6+9999999999")
				too_small_offset("#d6-9999999999")
			}
		}
	}
}
