use crate::{
	post::image_search::Provider,
	subs::{PartialAgent, SubManager, Subscribe},
	util,
};
use serde::{Deserialize, Serialize};
use yew::agent::AgentLink;

// Key used to store Options in local storage
const LS_KEY: &str = "options";

#[derive(Serialize, Deserialize, Clone, Copy, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ImageExpansionMode {
	None,
	FitWidth,
	FitHeight,
	FitScreen,
}

// Global user-set options
#[derive(Serialize, Deserialize, Clone)]
#[serde(default)]
pub struct Options {
	pub forced_anonymity: bool,
	pub relative_timestamps: bool,
	pub hide_thumbnails: bool,
	pub work_mode: bool,
	pub reveal_image_spoilers: bool,
	pub expand_gif_thumbnails: bool,
	pub enabled_image_search: Vec<Provider>,
	pub image_expansion_mode: ImageExpansionMode,
	pub audio_volume: u8,
}

impl Default for Options {
	fn default() -> Self {
		Self {
			forced_anonymity: false,
			relative_timestamps: true,
			hide_thumbnails: false,
			work_mode: false,
			reveal_image_spoilers: false,
			expand_gif_thumbnails: false,
			audio_volume: 100,
			image_expansion_mode: ImageExpansionMode::FitWidth,
			enabled_image_search: [
				Provider::Google,
				Provider::Yandex,
				Provider::IQDB,
				Provider::Trace,
				Provider::ExHentai,
			]
			.iter()
			.copied()
			.collect(),
		}
	}
}

// Partial subscription agent
#[derive(Default)]
pub struct PAgent();

impl PartialAgent for PAgent {
	type Data = Options;
	type Input = ();
	type Message = ();

	fn init(
		&mut self,
		_: &AgentLink<SubManager<Self>>,
		data: &mut Self::Data,
	) -> util::Result {
		// Read saved options, if any
		if let Some(v) = util::local_storage().get_item(LS_KEY)? {
			if let Ok(opts) = serde_json::from_str(&v) {
				*data = opts;
			}
		}
		Ok(())
	}
}

impl Subscribe for Options {
	type PA = PAgent;
}
