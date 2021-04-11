// TODO: Finish module and remove this
#![allow(unused)]

use crate::util::Result;
use common::payloads::post_body::EmbedProvider;
use serde::Deserialize;
use std::collections::HashMap;
use yew::{html, Component, ComponentLink, Html, Properties};

/// All the data required to render and expand an embed
struct Data {
	title: String,
	url: String,
	contents: String,
}

/// Describes and identifies a specific embed target
#[derive(std::hash::Hash, PartialEq, Eq)]
struct Descriptor {
	typ: EmbedProvider,
	target: String,
}

static mut CACHE: *mut HashMap<Descriptor, Data> = std::ptr::null_mut();

/// Render link to embedadble resource
pub fn render(p: EmbedProvider, url: &str) -> Html {
	html! {
		<span>{"TODO: embed rendering"}</span>
	}
}

/// Fetches and formats an embed's title and inner HTML content
trait Plugin: Default {
	/// Fetches embed data and send it to View over link
	fn fetch(&mut self, url: &str, link: ComponentLink<View>) -> Result;
}

struct View {
	// plugin: Box<dyn Plugin>,
	props: Props,
	link: ComponentLink<Self>,
}

enum Message {
	Hovered,
	Clicked,
	Fetch(Result<Data>),
}

#[derive(Clone, Properties, Eq, PartialEq)]
struct Props {
	pub provider: EmbedProvider,
	pub url: String,
}

impl Component for View {
	type Properties = Props;
	type Message = Message;

	// TODO: switch plugin on prop change

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			props,
			link,
			// TODO: match plugin via provider
			// plugin: Default::default(),
		}
	}

	fn change(&mut self, props: Self::Properties) -> bool {
		if self.props != props {
			self.props = props;
			// TODO: rematch plugin
			true
		} else {
			false
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		// TODO
		false
	}

	fn view(&self) -> Html {
		// TODO
		html! {
			<b>{"TODO: embeds"}</b>
		}
	}
}
