use crate::{comp_util, state::Change};
use yew::{html, Html};

#[derive(Default)]
pub struct Inner {
	expanded: bool,
}

pub type Menu = comp_util::HookedComponent<Inner>;

pub enum Message {
	Rerender,
	ToggleExpand,
}

impl comp_util::Inner for Inner {
	type Properties = super::common::Props;
	type Message = Message;

	#[inline]
	fn update_message() -> Self::Message {
		Message::Rerender
	}

	#[inline]
	fn subscribe_to(props: &Self::Properties) -> Vec<Change> {
		vec![Change::Post(props.id)]
	}

	#[inline]
	fn update(
		&mut self,
		_: &mut comp_util::Ctx<Self>,
		msg: Self::Message,
	) -> bool {
		match msg {
			Message::Rerender => true,
			Message::ToggleExpand => {
				self.expanded = !self.expanded;
				true
			}
		}
	}

	fn view(&self, c: &comp_util::Ctx<Self>) -> Html {
		let toggle = c.link().callback(|_| Message::ToggleExpand);

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
							<ul class="popup-menu glass no-select">
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
