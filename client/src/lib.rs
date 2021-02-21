// For html! macro
#![recursion_limit = "1024"]

#[macro_use]
mod lang;
#[macro_use]
mod banner;
#[macro_use]
mod util;
#[macro_use]
mod comp_util;
mod buttons;
mod connection;
mod mouse;
mod page_selector;
mod post;
mod state;
mod thread;
mod thread_index;
mod time;
mod user_bg;
mod widgets;

use common::debug_log;
use wasm_bindgen::prelude::*;
use yew::{html, Bridge, Bridged, Component, ComponentLink, Html};

struct App {
	link: ComponentLink<Self>,

	dragging: bool,

	// Keep here to load global state first and never drop the agents
	app_state: state::StateBridge,
	#[allow(unused)]
	mouse: Box<dyn Bridge<mouse::Agent>>,
}

enum Message {
	DraggingChange(bool),
	NOP,
}

impl Component for App {
	comp_no_props! {}
	type Message = Message;

	#[cold]
	fn create(_: Self::Properties, link: ComponentLink<Self>) -> Self {
		let mut s = Self {
			app_state: state::hook(&link, vec![], || Message::NOP),
			mouse: mouse::Agent::bridge(link.callback(|msg| match msg {
				mouse::Response::IsDragging(d) => Message::DraggingChange(d),
				_ => Message::NOP,
			})),
			dragging: false,
			link,
		};
		s.app_state.send(state::Request::FetchFeed(
			s.app_state.get().location.clone(),
		));

		// Static global event listeners. Put here to avoid overhead of spamming
		// a lot of event listeners and handlers on posts.
		util::add_static_listener(
			util::document(),
			"click",
			true,
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

				Message::NOP
			}),
		);

		s
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		match msg {
			Message::NOP => false,
			Message::DraggingChange(d) => {
				self.dragging = d;
				true
			}
		}
	}

	fn view(&self) -> Html {
		let mut cls = vec![];
		if self.dragging {
			cls.push("dragging");
		}

		html! {
			<section class=cls>
				<user_bg::Background />
				<div class="overlay-container">
					<banner::Banner />
					// z-index increases down
					<div class="overlay" id="post-form-overlay">
						<post::posting::PostForm
							id=self.app_state.get().open_post_id.unwrap_or(0)
						/>
					</div>
					<div class="overlay" id="modal-overlay">
						// TODO: modals
					</div>
					<div class="overlay" id="hover-overlay">
						// TODO: hover previews (post and image)
					</div>
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
	#[cfg(debug_assertions)]
	console_error_panic_hook::set_once();

	let (err1, err2) =
		futures::future::join(state::init(), lang::load_language_pack()).await;
	err1?;
	err2?;

	debug_log!("starting app");
	yew::start_app::<App>();

	Ok(())
}
