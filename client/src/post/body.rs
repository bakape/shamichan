use super::common::{Ctx, Message, PostComponent};
use crate::state;
use common::payloads::post_body::{Command, Node, PendingNode};
use std::fmt::Write;
use wasm_bindgen::JsCast;
use web_sys::{EventTarget, MouseEvent};
use yew::{html, Html};

// TODO: unit tests
pub fn render<'c, PC>(c: &Ctx<'c, PC>, n: &Node) -> Html
where
	PC: PostComponent + 'static,
{
	use Node::*;

	macro_rules! wrap_node {
		($tag:ident, $children:expr) => {
			html! {
				<$tag>{render(c, $children)}</$tag>
			}
		};
	}

	fn is_revealed(t: Option<EventTarget>) -> bool {
		t.map(|t| t.dyn_into::<web_sys::HtmlElement>().ok())
			.flatten()
			.map(|e| e.class_list().contains("reveal"))
			.unwrap_or(false)
	}

	match n {
		Empty => html! {},
		Text(s) => html! {s},
		Newline => html! { <br/> },
		Children(v) => v.iter().map(|n| render(c, n)).collect(),
		PostLink { id, thread, page } => {
			render_post_link(c, *id, *thread, *page)
		}
		Command(comm) => render_command(comm),
		URL(u) => html! {
			<a href=u.clone() target="_blank">{u}</a>
		},
		Reference { label, url } => html! {
			<a href=url.clone() target="_blank">{format!(">>>/{}/", label)}</a>
		},
		Embed { provider, url } => super::embeds::render(*provider, &url),
		Code(code) => {
			let el = crate::util::document().create_element("div").unwrap();
			el.set_inner_html(code);
			yew::virtual_dom::VNode::VRef(web_sys::Node::from(el))
		}
		Spoiler(ch) => html! {
			<del
				onclick=c.link().callback(|e: MouseEvent|
					Message::TextSpoilerInteraction{
						is_click: true,
						value: !is_revealed(e.target()),
					})
				onmouseenter=c.link().callback(|_|
					Message::TextSpoilerInteraction{
						is_click: false,
						value: true,
					})
				onmouseleave=c.link().callback(|_|
					Message::TextSpoilerInteraction{
						is_click: false,
						value: false,
					})
				class=if c.reveal_text_spoilers() { "reveal" } else { "" }
			>
				{render(c, ch)}
			</del>
		},
		Quoted(ch) => wrap_node!(em, ch),
		Bold(ch) => wrap_node!(b, ch),
		Italic(ch) => wrap_node!(i, ch),
		Pending(n) => render_pending_node(c, n),
	}
}

fn render_post_link<'c, PC>(
	c: &Ctx<'c, PC>,
	id: u64,
	thread: u64,
	page: u32,
) -> Html
where
	PC: PostComponent + 'static,
{
	// TODO: Attempt to look up link thread and page in global collection,
	// if not passed from server
	// TODO: Persist all single post fetches from server in global post
	// collection
	let mut extra = String::new();
	if match &c.app_state().location.feed {
		// If thread = 0, link has not had it's parenthood looked up yet on the
		// server
		state::FeedID::Thread { id: feed_id, .. } => {
			thread != 0 && feed_id != &thread
		}
		_ => true,
	} {
		extra += " ➡";
	}
	if c.app_state().mine.contains(&id) {
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
				onclick=c.link().callback(move |_| {
					// TODO:  Handle middle click

					state::navigate_to(state::Location{
						feed: state::FeedID::Thread{
							id: thread,
							page: page as i32,
						},
						focus: Some(state::Focus::Post(id)),
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
			if results.len() != 1 || offset != &0 {
				push!(" = {}", sum);
			}

			s
		}
	};

	html! {
		<strong>{inner}</strong>
	}
}

fn render_pending_node<'c, PC>(c: &Ctx<'c, PC>, n: &PendingNode) -> Html
where
	PC: PostComponent + 'static,
{
	use PendingNode::*;

	let inner = match n {
		Flip => "#flip ?".into(),
		EightBall => "#8ball ?".into(),
		Pyu => "#pyu ?".into(),
		PCount => "pcount ?".into(),
		Countdown(n) => format!("#countdown({})", n),
		Autobahn(n) => format!("#autobahn({})", n),
		PostLink(id) => match c.app_state().posts.get(id) {
			Some(p) => return render_post_link(c, *id, p.thread, p.page),
			None => format!(">>{}", id),
		},
		Dice {
			offset,
			faces,
			rolls,
		} => {
			let mut s = String::from("#");
			let sign = if offset < &0 { '-' } else { '+' };

			macro_rules! push {
				($format:expr, $($arg:expr),+) => {
					write!(&mut s, $format, $($arg),+).unwrap();
				};
			}

			if rolls > &1 {
				push!("{}", rolls);
			}
			push!("d{}", faces);
			if offset != &0 {
				push!("{}{}", sign, offset.abs());
			}

			for i in 0..*rolls {
				if i != 0 {
					s += " + ";
				}
				s += "?";
			}
			if offset != &0 {
				push!("{} {}", sign, offset.abs());
			}
			if rolls != &1 || offset != &0 {
				s += " = ?";
			}

			s
		}
	};

	html! {
		<strong class="pending">{inner}</strong>
	}
}
