// For html macro
#![recursion_limit = "1024"]

#[macro_use]
mod lang;
#[macro_use]
mod banner;
#[macro_use]
mod util;
#[macro_use]
mod comp_util;
#[macro_use]
mod agent_util;
mod buttons;
mod connection;
mod page_selector;
mod post;
mod state;
mod thread;
mod thread_index;
mod time;
mod user_bg;
mod widgets;

use protocol::debug_log;
use wasm_bindgen::prelude::*;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

struct App {
	link: ComponentLink<Self>,

	// Keep here to load global state first
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,
}

impl Component for App {
	comp_static! {}

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut a = state::Agent::bridge(link.callback(|_| ()));
		state::read(|s| {
			a.send(state::Request::FetchFeed(s.location.clone()));
		});

		let s = Self { state: a, link };

		// Static global event listeners. Put here to avoid overhead of spamming
		// a lot of event listeners and handlers on posts.
		util::add_static_listener(
			util::document(),
			"click",
			s.link.callback(|e: yew::events::MouseEvent| {
				util::with_logging(|| {
					use wasm_bindgen::JsCast;

					if let Some(el) = e
						.target()
						.map(|el| el.dyn_into::<web_sys::Element>().ok())
						.flatten()
					{
						if el.tag_name() == "DEL" {
							el.class_list().toggle("reveal")?;
						}
					}
					Ok(())
				});

				()
			}),
		);

		s
	}

	fn view(&self) -> Html {
		html! {
			<section>
				<user_bg::Background />
				<div class="overlay-container">
					<banner::Banner />
				</div>
				<section id="main">
					<widgets::AsideRow is_top=true />
					<hr />
					<thread_index::Threads />
					<hr />
					<widgets::AsideRow />
				</section>
			</section>
		}
	}
}

#[wasm_bindgen(start)]
pub async fn main_js() -> util::Result {
	if cfg!(debug_assertions) {
		console_error_panic_hook::set_once();
	}

	let (err1, err2) =
		futures::future::join(state::init(), lang::load_language_pack()).await;
	err1?;
	err2?;

	debug_log!("starting app");
	yew::start_app::<App>();

	Ok(())
}
