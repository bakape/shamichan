// For html macro
#![recursion_limit = "1024"]

#[macro_use]
mod lang;
#[macro_use]
mod banner;
mod connection;
mod state;
mod user_bg;
mod util;
mod widgets;

// #[macro_use]
// extern crate protocol;

use wasm_bindgen::prelude::*;
use yew::{html, Component, ComponentLink, Html};

struct App {}

impl Component for App {
	type Message = ();
	type Properties = ();

	fn create(_: Self::Properties, _: ComponentLink<Self>) -> Self {
		Self {}
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
					<section>{"TODO"}</section>
					<widgets::AsideRow />
				</section>
			</section>
		}
	}
}

#[wasm_bindgen(start)]
pub async fn main_js() -> util::Result {
	console_error_panic_hook::set_once();

	let s = state::get();
	s.feed = util::window().location().hash()?.parse().unwrap_or(0);
	s.load_auth_key()?;

	lang::load_language_pack().await?;
	yew::start_app::<App>();

	Ok(())
}
