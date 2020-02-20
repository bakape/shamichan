use yew::{html, Component, ComponentLink, Html};

pub struct Banner {}

impl Component for Banner {
	type Message = ();
	type Properties = ();

	fn create(_: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self {}
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
	}

	fn view(&self) -> Html {
		html! {
			<span id="banner" class="glass">
				<b id="banner-center" class="spaced"></b>
				<span>
					<super::connection::SyncCounter />
				</span>
			</span>
		}
	}
}
