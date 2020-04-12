use crate::state;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

pub struct Menu {
	link: ComponentLink<Self>,

	post: u64,

	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,

	expanded: bool,
}

pub enum Message {
	PostChange,
	ToggleExpand,
	NOP,
}

impl Component for Menu {
	type Message = Message;
	type Properties = super::Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		use state::{Agent, Response, Subscription};

		let mut s = Agent::bridge(link.callback(|u| match u {
			Response::NoPayload(Subscription::PostChange(_)) => {
				Message::PostChange
			}
			_ => Message::NOP,
		}));
		s.send(state::Request::Subscribe(state::Subscription::PostChange(
			props.id,
		)));
		Self {
			post: props.id,
			state: s,
			link,
			expanded: false,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::PostChange => true,
			Message::NOP => false,
			Message::ToggleExpand => {
				self.expanded = !self.expanded;
				true
			}
		}
	}

	fn view(&self) -> Html {
		html! {
			<a class="svg-link control">
				<svg
					onclick=self.link.callback(|_| Message::ToggleExpand)
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
