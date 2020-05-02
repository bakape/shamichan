use yew::{html, Component, ComponentLink, Html};

pub struct Banner {}

impl Component for Banner {
	crate::comp_static! {}

	fn create(_: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self {}
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
