use crate::time::scheduler::{RelativeTime, Response, Scheduler};
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

#[derive(Properties, Clone, PartialEq, Eq)]
pub struct Props {
	pub start: u32,
	pub end: u32,
}

pub struct Countdown {
	props: Props,
	current: RelativeTime,

	#[allow(unused)]
	scheduler: Box<dyn Bridge<Scheduler>>,
	#[allow(unused)]
	link: ComponentLink<Self>,
}

impl Component for Countdown {
	comp_prop_change! {Props}
	type Message = RelativeTime;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = Self {
			scheduler: Scheduler::bridge(link.callback(|u: Response| u.diff)),
			props,
			link,
			current: Default::default(),
		};
		s.scheduler.send(s.props.end);
		s
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		self.current = msg;
		true
	}

	fn view(&self) -> Html {
		html! {
			<strong>
				{format!(
					"#countdown({}) {}",
					self.props.end - self.props.start,
					self.current
				)}
			</strong>
		}
	}
}
