// For html macro
#![recursion_limit = "1024"]

#[macro_use]
mod lang;
#[macro_use]
mod banner;
mod buttons;
mod connection;
mod page_selector;
mod post;
mod state;
mod thread;
mod thread_index;
mod time;
mod user_bg;
mod util;
mod widgets;

use wasm_bindgen::prelude::*;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

struct App {
	// Keep here to load global state first
	#[allow(unused)]
	link: ComponentLink<Self>,
	#[allow(unused)]
	state: Box<dyn Bridge<state::Agent>>,
}

impl Component for App {
	type Message = ();
	type Properties = ();

	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut a = state::Agent::bridge(link.callback(|_| ()));
		state::read(|s| {
			a.send(state::Request::FetchFeed(s.location.clone()));
		});
		Self { state: a, link }
	}

	fn update(&mut self, _: Self::Message) -> bool {
		false
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

	state::init()?;
	lang::load_language_pack().await?;

	yew::start_app::<App>();

	Ok(())
}
