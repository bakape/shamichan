use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Default, Serialize, Deserialize)]
pub struct SpamScores {
	// Score per unicode character for any post body modification
	pub character: usize,

	// Score for inserting an image into the post
	pub image: usize,

	// Score for creating a post
	pub post_creation: usize,
}

// Global server configurations
#[derive(Default, Serialize, Deserialize)]
pub struct Config {
	// Enable captchas and antispam
	pub captcha: bool,

	// Configured labeled links to resources
	pub links: HashMap<String, String>,

	// Amounts to increase spam score by for a user action
	pub spam_scores: SpamScores,
}

protocol::gen_global!(pub, pub, Config);
