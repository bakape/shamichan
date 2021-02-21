use crate::{comp_util, state};
use serde::{Deserialize, Serialize};
use yew::{html, Html, Properties};

// TODO: rewrite to a simpler `<a>1</a> <b>2<b> <a>3</a>` page selector with
// only 3 pages displayed at a time and a <select> at the end to navigate to
// a specific page

#[derive(Default)]
pub struct Inner {
	offset: u32,
	page_count: u32,
}

/// Used to select a certain page of a thread
pub type PageSelector = comp_util::HookedComponent<Inner>;

#[derive(Clone, Properties, Eq, PartialEq, Debug)]
pub struct Props {
	pub thread: u64,
}

#[derive(Serialize, Deserialize, Clone)]
pub enum Message {
	Scroll { left: bool, to_end: bool },
	SelectPage(u32),
	ThreadUpdate,
	NOP,
}

impl comp_util::Inner for Inner {
	type Message = Message;
	type Properties = Props;

	fn init(&mut self, c: &mut comp_util::Ctx<Self>) {
		self.fetch_page_count(c);
	}

	fn update_message() -> Self::Message {
		Message::ThreadUpdate
	}

	fn subscribe_to(props: &Self::Properties) -> Vec<state::Change> {
		vec![state::Change::Thread(props.thread)]
	}

	fn update(
		&mut self,
		c: &mut comp_util::Ctx<Self>,
		msg: Self::Message,
	) -> bool {
		use Message::*;

		match msg {
			Scroll { left, to_end } => {
				let old = self.offset;
				let max = if self.page_count > 5 {
					self.page_count - 5
				} else {
					0
				};

				if to_end {
					self.offset = if left { 0 } else { max };
				} else {
					if left {
						if self.offset > 0 {
							self.page_count -= 1;
						}
					} else {
						if self.offset < max {
							self.offset += 1;
						}
					}
				}

				self.offset != old
			}
			SelectPage(_) => todo!("page navigation"),
			ThreadUpdate => {
				let old = self.page_count;
				self.fetch_page_count(c);
				old != self.page_count
			}
			NOP => false,
		}
	}

	fn view(&self, c: &comp_util::Ctx<Self>) -> Html {
		html! {
			<span class="spaced mono no-select">
				{self.render_scroll_button(&c, "<<", Message::Scroll{
					left: true,
					to_end: true,
				})}
				{
					if self.page_count > 5 {
						self.render_scroll_button(&c, "<", Message::Scroll{
							left: true,
							to_end: false,
						})
					} else {
						html! {}
					}
				}
				{
					for (self.offset..self.page_count).map(|i| html! {
						<a
							onclick=c.link().callback(move |_|
								Message::SelectPage(i)
							)
						>
							{i}
						</a>
					})
				}
				{
					if self.page_count > 5 {
						self.render_scroll_button(&c, ">", Message::Scroll{
							left: false,
							to_end: false,
						})
					} else {
						html! {}
					}
				}
				{self.render_scroll_button(&c, ">>", Message::Scroll{
					left: false,
					to_end: true,
				})}
			</span>
		}
	}
}

impl Inner {
	fn render_scroll_button(
		&self,
		c: &comp_util::Ctx<Self>,
		text: &str,
		msg: Message,
	) -> Html {
		html! {
			<a onclick=c.link().callback(move |_| msg.clone())>{text}</a>
		}
	}

	/// Fetch and set new page count value for thread from global state
	fn fetch_page_count(&mut self, c: &mut comp_util::Ctx<Self>) {
		self.page_count = c
			.app_state()
			.threads
			.get(&c.props().thread)
			.map(|t| t.page_count)
			.unwrap_or(1);
	}
}
