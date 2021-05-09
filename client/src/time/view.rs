use super::scheduler::{Response, Scheduler};
use js_sys::Date;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

/// Central thread container
pub struct Time {
	#[allow(unused)]
	link: ComponentLink<Self>,

	props: Props,

	current: Response,
	scheduler: Box<dyn Bridge<Scheduler>>,
}

#[derive(Clone, Properties, Eq, PartialEq)]
pub struct Props {
	pub time: u32,
}

impl Component for Time {
	type Message = Response;
	type Properties = Props;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = Time {
			scheduler: Scheduler::bridge(link.callback(|u| u)),
			props,
			link,
			current: Default::default(),
		};
		s.scheduler
			.send(super::scheduler::Request::Register(s.props.time));
		s
	}

	fn update(&mut self, new: Self::Message) -> bool {
		self.current = new;
		true
	}

	fn change(&mut self, props: Self::Properties) -> bool {
		if self.props != props {
			self.props = props;
			self.scheduler
				.send(super::scheduler::Request::ChangeTimestamp(
					self.props.time,
				));
			true
		} else {
			false
		}
	}

	fn view(&self) -> Html {
		// Placeholder post
		if self.props.time == 0 {
			return html! {};
		}

		static MONTHS: [&str; 12] = [
			"january",
			"february",
			"march",
			"april",
			"may",
			"june",
			"july",
			"august",
			"september",
			"october",
			"november",
			"december",
		];
		static DAYS: [&str; 7] = [
			"sunday",
			"monday",
			"tuesday",
			"wednesday",
			"thursday",
			"friday",
			"saturday",
		];

		let d = Date::new(&(self.props.time as f64 * 1000.0).into());
		let abs = format!(
			"{:0>2} {} {} ({}) {:0>2}:{:0>2}:{:0>2}",
			d.get_date(),
			localize!(MONTHS[d.get_month() as usize]),
			d.get_full_year(),
			localize!(DAYS[d.get_day() as usize]),
			d.get_hours(),
			d.get_minutes(),
			d.get_seconds(),
		);

		let rel = self.current.diff.to_string();

		html! {
			<time
				title=if self.current.use_relative {
					&abs
				} else {
					&rel
				}
			>
				{
					if self.current.use_relative {
						&rel
					} else {
						&abs
					}
				}
			</time>
		}
	}
}
