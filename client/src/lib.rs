#[macro_use]
mod lang;
mod connection;
mod state;
mod util;

// #[macro_use]
// extern crate protocol;

use wasm_bindgen::prelude::*;
use yew::{html, Component, ComponentLink, Html};

struct UserBackground {}

impl Component for UserBackground {
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
			<>
				<div id="user-background"></div>
			</>
		}
	}
}

struct Banner {}

impl Component for Banner {
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
			<>
				<span id="banner" class="glass">
					<b id="banner-center" class="spaced"></b>
					<span>
						<connection::SyncCounter />
					</span>
				</span>
			</>
		}
	}
}

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
				<UserBackground />
				<div class="overlay-container">
					<Banner />
				</div>
			</section>
		}
	}
}

#[wasm_bindgen(start)]
pub async fn main_js() -> util::Result {
	console_error_panic_hook::set_once();

	async fn run(s: &mut state::State) -> util::Result {
		s.thread = util::window().location().hash()?.parse().unwrap_or(0);
		s.load_auth_key()?;
		lang::load_language_pack().await?;
		yew::start_app::<App>();
		Ok(())
	}

	state::with(run).await?;

	Ok(())
}
