use super::common::{Message, PostComponent, RenderCtx};
use crate::state;
use protocol::payloads::post_body::{Command, Node, PostLink};
use std::fmt::Write;
use yew::{html, Html};

pub fn render<'s, 'c, PC>(c: &RenderCtx<'s, 'c, PC>) -> Html
where
	PC: PostComponent + 'static,
{
	match &c.post.body {
		Some(n) => render_node(c, n),
		None => html! {},
	}
}

fn render_node<'s, 'c, PC>(c: &RenderCtx<'s, 'c, PC>, n: &Node) -> Html
where
	PC: PostComponent + 'static,
{
	macro_rules! wrap_children {
		($tag:ident, $children:expr) => {
			html! {
				<$tag>
					{
						for $children.iter().map(|ch| render_node(c, ch))
					}
				</$tag>
			}
		};
	}

	match n {
		Node::Text(s) => html! {s},
		Node::PostLink(l) => render_post_link(c, l.clone()),
		Node::Command(comm) => render_command(comm),
		Node::URL(u) => html! {
			<a href=u.clone()>{u}</a>
		},
		Node::Reference { label, url } => html! {
			<a href=url.clone()>{format!(">>>/{}/", label)}</a>
		},
		Node::Embed { .. } => todo!(),
		Node::Code(_) => todo!(),
		Node::Spoiler(ch) => wrap_children!(del, ch),
		Node::Quoted(ch) => wrap_children!(em, ch),
		Node::Bold(ch) => wrap_children!(b, ch),
		Node::Italic(ch) => wrap_children!(i, ch),
	}
}

fn render_post_link<'s, 'c, PC>(c: &RenderCtx<'s, 'c, PC>, l: PostLink) -> Html
where
	PC: PostComponent + 'static,
{
	let mut extra = String::new();
	if match &c.app.location.feed {
		state::FeedID::Thread { id, .. } => id != &c.post.thread,
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
	let inner = match comm {
		Command::AutoBahn(hours) => format!("#autobahn({})", hours),
		Command::EightBall(msg) => format!("#8ball {}", msg),
		Command::Flip(b) => {
			format!("#flip {}", if *b { "flap" } else { "flop" })
		}
		Command::Countdown { start, end } => {
			return html! {
				<super::countdown::Countdown start=start end=end />
			}
		}
		Command::Dice {
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
