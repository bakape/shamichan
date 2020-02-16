use crate::lang::pluralize;
use js_sys::Date;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html, Properties};

// Central thread container
pub struct View {
	link: ComponentLink<Self>,
	time: u32,
	relative: bool,
}

pub enum Message {
	Tick,
	RelativeChanged(bool),
}

#[derive(Clone, Properties)]
pub struct Props {
	pub time: u32,
	pub relative: bool,
}

impl Component for View {
	type Message = Message;
	type Properties = Props;

	fn create(p: Self::Properties, link: ComponentLink<Self>) -> Self {
		// TODO: Link to agent
		// TODO: Have agent send first tick on creation
		Self {
			time: p.time,
			relative: p.relative,
			link: link,
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::Tick => (),
			Message::RelativeChanged(r) => {
				self.relative = r;
			}
		}
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

		let rel = self.format_relative();

		html! {
			<>
				<time
					title=if self.relative {
						&abs
					} else {
						&rel
					}
				>
					{
						if self.relative {
							&rel
						} else {
							&abs
						}
					}
				</time>
			</>
		}
	}
}

impl View {
	fn format_relative(&self) -> String {
		let now = (Date::now() / 1000.0) as i64;
		let mut time = (now - self.time as i64) / 60;
		let mut is_future = false;
		if time < 0 {
			time = -time;
			is_future = true;
		}

		macro_rules! format {
			($unit:expr) => {
				return localize!(
					if is_future {
						"time_in"
					} else {
						"time_ago"
					},
					{
						"number" => &time.to_string()
						"unit" => pluralize($unit, time)
					}
				);
			};
		}

		static UNITS: [(&str, i64); 4] =
			[("minute", 60), ("hour", 24), ("day", 30), ("month", 12)];
		for u in UNITS.iter() {
			if time < u.1 {
				format!(u.0);
			}
			time /= u.1;
		}

		format!("year")
	}
}
