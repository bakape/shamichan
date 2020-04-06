use crate::{
	subs::{PartialAgent, SubManager, Subscribe},
	util,
};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use yew::agent::AgentLink;

// Exported public server configurations
//
// TODO: Get config updates though websocket
#[derive(Serialize, Deserialize, Default, Clone)]
pub struct Configs {
	pub captcha: bool,
	pub mature: bool,
	pub prune_threads: bool,
	pub thread_expiry: u32,
	pub max_size: u64,
	pub default_lang: String,
	pub default_theme: String,
	pub image_root_override: String,
	pub links: HashMap<String, String>,
}

// Partial subscription agent
#[derive(Default)]
pub struct PAgent();

impl PartialAgent for PAgent {
	type Data = Configs;
	type Input = ();
	type Message = ();

	fn init(
		&mut self,
		_: &AgentLink<SubManager<Self>>,
		data: &mut Self::Data,
	) -> util::Result {
		// Read configs from JSON embedded in the HTML
		*data = serde_json::from_str(
			&util::document()
				.get_element_by_id("config-data")
				.ok_or("inline configs not found")?
				.inner_html(),
		)?;
		Ok(())
	}
}

impl Subscribe for Configs {
	type PA = PAgent;
}
