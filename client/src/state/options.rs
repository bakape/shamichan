use crate::{post::image_search::Provider, util};
use serde::{Deserialize, Serialize};

/// Key used to store Options in local storage
const OPTIONS_KEY: &str = "options";

#[derive(Serialize, Deserialize, Clone, Copy, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum ImageExpansionMode {
	None,
	FitWidth,
	FitHeight,
	FitScreen,
}

/// Global user-set options
#[derive(Serialize, Deserialize)]
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

impl Options {
	/// Read saved options, if any
	pub fn load(&mut self) {
		if let Some(v) = util::local_storage().get_item(OPTIONS_KEY).unwrap() {
			if let Ok(opts) = serde_json::from_str(&v) {
				*self = opts;
			}
		}
	}
}
