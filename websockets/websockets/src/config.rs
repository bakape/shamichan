// Global server configurations
#[derive(Default)]
pub struct Config {
	pub captcha: bool,
}

super::gen_global_rwlock!(pub, Config);
