use crate::time::view::{Props, Time};
use yew::{html, Component, ComponentLink, Html};

pub struct Countdown {
	props: Props,
}

impl Component for Countdown {
	comp_prop_change! {Props}
	comp_no_update! {}

	fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self { props }
	}

	fn view(&self) -> Html {
		html! {
			<strong>
				<Time time=self.props.time />
			</strong>
		}
	}
}
