use clap::Clap;
use serde::{Deserialize, Serialize};

// TODO: read configs from DB or fallback to default, if none

lazy_static::lazy_static! {
	/// Configurations for this specific application server
	pub static ref SERVER: Server = Server::parse();
}

/// Configurations for this specific application server
#[derive(Clap)]
pub struct Server {
	/// Database address to connect to
	#[clap(short, long)]
	pub database: String,

	/// Address for the server to listen on
	#[clap(short, long, default_value = "127.0.0.1:8000")]
	pub address: String,

	/// Indicates this server is behind a reverse proxy and can honour
	/// X-Forwarded-For and similar headers
	#[clap(short, long)]
	pub reverse_proxied: bool,
}

/// Antispam scores for various client actions
#[derive(Serialize, Deserialize)]
pub struct SpamScores {
	/// Score per unicode character for any post body modification
	pub character: usize,

	/// Score for inserting an image into the post
	pub image: usize,

	/// Score for creating a post
	pub post_creation: usize,
}

impl Default for SpamScores {
	fn default() -> Self {
		Self {
			character: 85,
			image: 7500,
			post_creation: 7500,
		}
	}
}

/// Global server configurations
#[derive(Serialize, Deserialize, Default)]
pub struct Config {
	/// Global server configurations exposed to the client
	pub public: common::config::Public,

	/// Instruct bots to not access the site
	pub disable_robots: bool,

	/// Antispam scores for various client actions
	pub spam_scores: SpamScores,

	/// Booru tags for the captcha pool
	pub captcha_tags: Vec<String>,
}

common::gen_global!(
	// Global configurations
	Config {
		pub fn read();
		pub fn write();
	}
);
