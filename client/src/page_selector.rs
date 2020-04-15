use crate::state;
use serde::{Deserialize, Serialize};
use yew::{html, Component, ComponentLink, Html, Properties};

// Used to select a certain page of a thread
pub struct PageSelector {
	#[allow(unused)]
	bridge: state::HookBridge,

	link: ComponentLink<Self>,

	thread: u64,
	offset: u32,
	page_count: u32,
}

#[derive(Clone, Properties)]
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

impl Component for PageSelector {
	type Message = Message;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = Self {
			bridge: state::hook(
				&link,
				&[state::Change::Thread(props.thread)],
				|_| Message::ThreadUpdate,
			),
			thread: props.thread,
			link,
			offset: 0,
			page_count: 0,
		};
		s.fetch_page_count();
		s
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::Scroll { left, to_end } => {
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
			Message::SelectPage(_) => todo!("page navigation"),
			Message::ThreadUpdate => {
				let old = self.page_count;
				self.fetch_page_count();
				old != self.page_count
			}
			Message::NOP => false,
		}
	}

	fn view(&self) -> Html {
		html! {
			<span class="spaced mono no-select">
				{self.render_scroll_button("<<", Message::Scroll{
					left: true,
					to_end: true,
				})}
				{
					if self.page_count > 5 {
						self.render_scroll_button("<", Message::Scroll{
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
							onclick=self.link.callback(move |_|
								Message::SelectPage(i)
							)
						>
							{i}
						</a>
					})
				}
				{
					if self.page_count > 5 {
						self.render_scroll_button(">", Message::Scroll{
							left: false,
							to_end: false,
						})
					} else {
						html! {}
					}
				}
				{self.render_scroll_button(">>", Message::Scroll{
					left: false,
					to_end: true,
				})}
			</span>
		}
	}
}

impl PageSelector {
	fn render_scroll_button(&self, text: &str, msg: Message) -> Html {
		html! {
			<a onclick=self.link.callback(move |_| msg.clone())>{text}</a>
		}
	}

	// Fetch and set new page count value for thread from global state
	fn fetch_page_count(&mut self) {
		self.page_count = state::read(|s| {
			s.threads
				.get(&self.thread)
				.map(|t| t.last_page + 1)
				.unwrap_or(1)
		});
	}
}
