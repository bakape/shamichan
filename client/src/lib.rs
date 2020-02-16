// For html macro
#![recursion_limit = "1024"]

#[macro_use]
mod lang;
#[macro_use]
mod banner;
mod connection;
mod state;
mod threads;
mod user_bg;
mod util;
mod widgets;

// #[macro_use]
// extern crate protocol;

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
		let mut s = state::Agent::bridge(link.callback(|_| ()));
		s.send(state::Request::FetchFeed {
			id: state::get().feed,
			sync: false,
		});
		Self {
			state: s,
			link: link,
		}
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
					<threads::Threads />
					<widgets::AsideRow />
				</section>
			</section>
		}
	}
}

#[wasm_bindgen(start)]
pub async fn main_js() -> util::Result {
	console_error_panic_hook::set_once();

	state::init();
	lang::load_language_pack().await?;

	yew::start_app::<App>();

	Ok(())
}
