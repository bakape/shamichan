use clap::Clap;
use serde::{Deserialize, Serialize};
use std::sync::{Arc, RwLock};

// TODO: read configs from DB or fallback to default, if none

lazy_static::lazy_static! {
	/// Configurations for this specific application server
	pub static ref SERVER: Server = Server::parse();

	/// Global configurations. Wrapped for swapping whole.
	static ref CONFIG: RwLock<Arc<Config>> = Default::default();
}

/// Configurations for this specific application server
#[derive(Clap)]
pub struct Server {
	/// Database address to connect to
	#[clap(short, long, env = "DATABASE_URL")]
	pub database: String,

	/// Address for the server to listen on
	#[clap(short, long, default_value = "127.0.0.1:8000", env = "ADDRESS")]
	pub address: String,

	/// Indicates this server is behind a reverse proxy and can honour
	/// X-Forwarded-For and similar headers
	#[clap(short, long, env = "REVERSE_PROXIED")]
	pub reverse_proxied: bool,
}

/// Antispam scores for various client actions
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct SpamScores {
	/// Score per unicode character for any post body modification
	pub character: usize,

	/// Score for inserting an image into the post
	pub image: usize,

	/// Score for creating a post
	pub post_creation: usize,
}

impl Default for SpamScores {
	#[inline]
	fn default() -> Self {
		Self {
			character: 85,
			image: 7500,
			post_creation: 7500,
		}
	}
}

/// Global server configurations
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Config {
	/// Global server configurations exposed to the client.
	///
	/// Wrapped in it's own Arc to be passable around without copying.
	pub public: Arc<common::config::Public>,

	/// Instruct bots to not access the site
	pub disable_robots: bool,

	/// Antispam scores for various client actions
	pub spam_scores: SpamScores,

	/// Booru tags for the captcha pool
	pub captcha_tags: Vec<String>,
}

impl Default for Config {
	#[inline]
	fn default() -> Self {
		Self {
			public: Default::default(),
			disable_robots: Default::default(),
			spam_scores: Default::default(),
			captcha_tags: vec![
				"patchouli_knowledge".into(),
				"cirno".into(),
				"hakurei_reimu".into(),
			],
		}
	}
}

/// Get a snapshot of the current configuration
#[inline]
pub fn get() -> Arc<Config> {
	CONFIG.read().unwrap().clone()
}

/// Set the configurations to a new value
#[cold]
pub fn set(c: Config) {
	// TODO: send new configs to all clients
	let c = Arc::new(c);
	*CONFIG.write().unwrap() = c;
}
