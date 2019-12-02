use std::error::Error;
use std::net::IpAddr;

// Maps to a websocket client on the Go side
pub struct Client {
	id: u64,
	ip: IpAddr,
}

impl Client {
	pub fn new(id: u64, ip: IpAddr) -> Self {
		Self { id: id, ip: ip }
	}

	// Handle received message
	pub fn receive_message(
		&mut self,
		msg: &[u8],
	) -> Result<(), Box<dyn Error>> {
		unimplemented!("message processing")
	}
}
