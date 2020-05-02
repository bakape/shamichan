use crate::state::{hook, Change, HookBridge};
use yew::{html, Component, ComponentLink, Html};

pub struct Menu {
	post: u64,
	expanded: bool,

	link: ComponentLink<Self>,
	#[allow(unused)]
	hook: HookBridge,
}

pub enum Message {
	Rerender,
	ToggleExpand,
}

impl Component for Menu {
	type Message = Message;
	type Properties = super::Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			post: props.id,
			// Not used right now, but will be needed for menu items
			hook: hook(&link, &[Change::Post(props.id)], |_| Message::Rerender),
			link,
			expanded: false,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::Rerender => true,
			Message::ToggleExpand => {
				self.expanded = !self.expanded;
				true
			}
		}
	}

	fn view(&self) -> Html {
		let toggle = self.link.callback(|_| Message::ToggleExpand);

		html! {
			<a class="svg-link control" onclick=toggle.clone()>
				<svg
					onclick=toggle
					xmlns="http://www.w3.org/2000/svg"
					width="8"
					height="8"
					viewBox="0 0 8 8"
				>
					<path
						d="M1.5 0l-1.5 1.5 4 4 4-4-1.5-1.5-2.5 2.5-2.5-2.5z"
						transform="translate(0 1)"
					/>
				</svg>
				{
					if self.expanded {
						html! {
							<ul class="popup-menu glass">
								<li>{"TODO"}</li>
							</ul>
						}
					} else {
						html! {}
					}
				}
			</a>
		}
	}
}
