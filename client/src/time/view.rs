use super::scheduler::{Response, Scheduler};
use js_sys::Date;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct View {
	#[allow(unused)]
	link: ComponentLink<Self>,

	time: u32,
	current: Response,
	scheduler: Box<dyn Bridge<Scheduler>>,
}

#[derive(Clone, Properties)]
pub struct Props {
	pub time: u32,
}

impl Component for View {
	type Message = Response;
	type Properties = Props;

	fn create(p: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = Self {
			scheduler: Scheduler::bridge(link.callback(|u| u)),
			time: p.time,
			link,
			current: Default::default(),
		};
		s.scheduler.send(s.time);
		s
	}

	fn update(&mut self, new: Self::Message) -> bool {
		self.current = new;
		true
	}

	fn view(&self) -> Html {
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

		let d = Date::new(&(self.time as f64 * 1000.0).into());
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
