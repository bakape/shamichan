// TODO: Finish module and remove this
#![allow(unused)]

use crate::util::Result;
use common::payloads::post_body::{Embed, EmbedProvider};
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
pub fn render(e: Embed) -> Html {
	html! {
		<span>{"TODO: embed rendering"}</span>
	}
}

/// Fetches and formats an embed's title and inner HTML content
trait Fetch: Default {
	/// Fetches embed data and send it to View over link
	fn fetch(&mut self, e: Embed, link: ComponentLink<View<Self>>) -> Result;
}

struct View<F>
where
	F: Fetch + 'static,
{
	fetcher: F,
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
	pub embed: Embed,
}

impl<F> Component for View<F>
where
	F: Fetch + 'static,
{
	comp_prop_change! {Props}
	type Message = Message;

	fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
		Self {
			props,
			link,
			fetcher: Default::default(),
		}
	}

	fn update(&mut self, msg: Self::Message) -> bool {
		todo!()
	}

	fn view(&self) -> Html {
		todo!()
	}
}
