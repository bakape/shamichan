use yew::{html, Component, ComponentLink, Html};

pub struct Background {}

impl Component for Background {
	comp_static!{}

	fn create(_: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self {}
	}

	fn view(&self) -> Html {
		html! {
			<div id="user-background"></div>
		}
	}
}
