use crate::{
	comp_util,
	state::{self, navigate_to, FeedID, Focus, Location},
};
use yew::{html, ChangeData, Html, Properties};

// TODO: rewrite to a simpler `<a>1</a> <b>2<b> <a>3</a>` page selector with
// only 3 pages displayed at a time and a <select> at the end to navigate to
// a specific page

#[derive(Default)]
pub struct Inner;

/// Used to select a certain page of a thread
pub type PageSelector = comp_util::HookedComponent<Inner>;

#[derive(Clone, Properties, Eq, PartialEq, Debug)]
pub struct Props {
	pub thread: u64,
}

impl comp_util::HookedComponentInner for Inner {
	type Message = ();
	type Properties = Props;

	#[inline]
	fn update_message() -> Self::Message {
		()
	}

	#[inline]
	fn subscribe_to(props: &Self::Properties) -> Vec<state::Change> {
		use state::Change::*;

		vec![Thread(props.thread), Location]
	}

	#[inline]
	fn update(
		&mut self,
		_: &mut comp_util::Ctx<Self>,
		_: Self::Message,
	) -> bool {
		true
	}

	fn view(&self, c: &comp_util::Ctx<Self>) -> Html {
		let current = match &c.app_state().location.feed {
			FeedID::Thread { id, page } if id == &c.props().thread => {
				*page as u32
			}
			_ => 0,
		};
		let page_count = c.app_state().page_count(&c.props().thread);
		let thread = c.props().thread;

		html! {
			<select
				onchange=c.link().callback(move |ch: ChangeData| match ch {
					ChangeData::Select(el) => {
						if let Ok(page) = el.value().parse::<u32>() {
							navigate_to(
								Location{
									feed: FeedID::Thread{
										id: thread,
										page: page as i32,
									},
									focus: if page == 0 {
										Some(Focus::Top)
									} else if page == page_count - 1 {
										Some(Focus::Bottom)
									} else {
										None
									},
								}
							);
						}
					}
					_ => ()
				})
			>
				{
					for (0..page_count).map(|i| html! {
						<option value=i selected=i == current>{i}</option>
					})
				}
			</select>
		}
	}
}
