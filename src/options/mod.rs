#![allow(dead_code)] // TEMP

use externs::local_storage;
use serde_json;
use std::default::Default;

#[allow(non_snake_case)]
#[derive(Deserialize)]
pub struct Options {
	hideThumbs: bool,
	imageHover: bool,
	webmHover: bool,
	autogif: bool,
	spoilers: bool,
	notification: bool,
	anonymise: bool,
	postInlineExpand: bool,
	relativeTime: bool,
	nowPlaying: bool,
	illyaDance: bool,
	illyaDanceMute: bool,
	horizontalPosting: bool,
	hideRecursively: bool,
	replyRight: bool,
	workModeToggle: bool,
	userBG: bool,
	customCSSToggle: bool,
	mascot: bool,
	alwaysLock: bool,
	newPost: u32,
	toggleSpoiler: u32,
	done: u32,
	expandAll: u32,
	workMode: u32,
	inlineFit: String,
	theme: String,
	customCSS: String,
}

impl Default for Options {
	fn default() -> Options {
		Options {
			hideThumbs: false,
			imageHover: true,
			webmHover: false,
			autogif: false,
			spoilers: true,
			notification: true,
			anonymise: false,
			postInlineExpand: true,
			relativeTime: false,
			nowPlaying: false,
			illyaDance: false,
			illyaDanceMute: false,
			horizontalPosting: false,
			hideRecursively: false,
			replyRight: false,
			workModeToggle: false,
			userBG: false,
			customCSSToggle: false,
			mascot: false,
			alwaysLock: false,
			newPost: 78,
			toggleSpoiler: 73,
			done: 83,
			expandAll: 69,
			workMode: 66,
			inlineFit: String::from("width"),
			theme: String::from("ashita"), // TODO: Read from configs
			customCSS: String::new(),
		}
	}
}

fn load() -> Options {
	let s = local_storage::get("options");
	if s.is_empty() {
		return Options::default();
	}
	match serde_json::from_str::<Options>(&s) {
		Ok(opts) => opts,
		_ => Options::default(),
	}
}
