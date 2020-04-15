// Global server configurations
#[derive(Default)]
pub struct Config {
	pub captcha: bool,
}

protocol::gen_global!(pub, pub, Config);
