static mut SERVER_CONF: *const ServerConfig = std::ptr::null();

#[derive(serde::Deserialize, Default)]
pub struct ServerConfig {
	pub listening_address: String,
}

// Load server configuration from file
pub fn load_server_config(
) -> Result<&'static ServerConfig, Box<dyn std::error::Error>> {
	let c: ServerConfig =
		serde_json::from_reader(std::fs::File::open("config.json")?)?;
	unsafe { SERVER_CONF = Box::into_raw(Box::new(c)) };
	Ok(server_config())
}

// Returns configuration of this particular server.
// Must be called only after load_server_config() has been called once.
pub fn server_config() -> &'static ServerConfig {
	unsafe { std::mem::transmute(SERVER_CONF) }
}
