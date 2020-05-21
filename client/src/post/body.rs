use super::common::{Message, PostComponent, RenderCtx};
use crate::state;
use protocol::payloads::post_body::{Command, Node, PostLink};
use std::fmt::Write;
use yew::{html, Html};

pub fn render<'s, 'c, PC>(c: &RenderCtx<'s, 'c, PC>, n: &Node) -> Html
where
	PC: PostComponent + 'static,
{
	use Node::*;

	macro_rules! wrap_node {
		($tag:ident, $children:expr) => {
			html! {
				<$tag>{render(c, ch)}</$tag>
			}
		};
	}

	match n {
		Empty => html! {},
		Text(s) => html! {s},
		NewLine => html! { <br/> },
		Siblings([l, r]) => html! {
			<>
				{render(c, &*l)}
				{render(c, &*r)}
			</>
		},
		PostLink(l) => render_post_link(c, l.clone()),
		Command(comm) => render_command(comm),
		URL(u) => html! {
			<a href=u.clone() target="_blank">{u}</a>
		},
		Reference { label, url } => html! {
			<a href=url.clone() target="_blank">{format!(">>>/{}/", label)}</a>
		},
		Embed(e) => super::embeds::render(e.clone()),
		Code(code) => {
			let el = crate::util::document().create_element("div").unwrap();
			el.set_inner_html(code);
			yew::virtual_dom::VNode::VRef(web_sys::Node::from(el))
		}
		Spoiler(ch) => wrap_node!(del, ch),
		Quoted(ch) => wrap_node!(em, ch),
		Bold(ch) => wrap_node!(b, ch),
		Italic(ch) => wrap_node!(i, ch),

		// Must not happen
		Pending(_) => html! {
			<b class="admin">{"ERROR: PENDING NODE SENT TO CLIENT"}</b>
		},
	}
}

fn render_post_link<'s, 'c, PC>(c: &RenderCtx<'s, 'c, PC>, l: PostLink) -> Html
where
	PC: PostComponent + 'static,
{
	// TODO: Attempt to look up link thread and page in global collection,
	// if not passed from server
	// TODO: Persist all single post fetches from server in global post
	// collection

	let mut extra = String::new();
	if match &c.app.location.feed {
		// If thread = 0, link has not had it's parenthood looked up yet on the
		// server
		state::FeedID::Thread { id, .. } => l.thread != 0 && id != &l.thread,
		_ => true,
	} {
		extra += " ➡";
	}
	if c.app.mine.contains(&l.id) {
		extra.push(' ');
		extra += localize!("you");
	}

	html! {
		// TODO: Hover preview on both
		<>
			// TODO: inline post on click
			<a>
				{
					if !extra.is_empty() {
						html! {extra}
					} else {
						html! {}
					}
				}
			</a>
			<a
				onclick=c.link.callback(move |_| {
					// TODO:  Handle middle click

					state::navigate_to(state::Location{
						feed: state::FeedID::Thread{
							id: l.thread,
							page: l.page as i32,
						},
						focus: Some(state::Focus::Post(l.id)),
					});
					Message::NOP
				})
			>
				{" #"}
			</a>
		</>
	}
}

fn render_command(comm: &Command) -> Html {
	use Command::*;

	let inner = match comm {
		Countdown { start, secs } => {
			return html! {
				<super::countdown::Countdown start=start end=start+secs />
			}
		}
		Autobahn(hours) => format!("#autobahn({})", hours),
		EightBall(msg) => format!("#8ball {}", msg),
		Flip(b) => format!("#flip {}", if *b { "flap" } else { "flop" }),
		Pyu(n) => format!("#pyu {}", n),
		PCount(n) => format!("#pcount {}", n),
		Dice {
			offset,
			faces,
			results,
		} => {
			let mut s = String::from("#");
			let sign = if offset < &0 { '-' } else { '+' };

			macro_rules! push {
				($format:expr, $($arg:expr),+) => {
					write!(&mut s, $format, $($arg),+).unwrap();
				};
			}

			if results.len() > 1 {
				push!("{}", results.len());
			}
			push!("d{}", faces);
			if offset != &0 {
				push!("{}{}", sign, offset.abs());
			}

			let mut sum = 0_i32;
			for (i, r) in results.iter().enumerate() {
				if i != 0 {
					s += " + ";
				}
				sum += *r as i32;
				push!("{}", r);
			}
			if offset != &0 {
				sum += *offset as i32;
				push!("{} {}", sign, offset.abs());
			}
			push!(" = {}", sum);

			s
		}
	};

	html! {
		<strong>{inner}</strong>
	}
}
