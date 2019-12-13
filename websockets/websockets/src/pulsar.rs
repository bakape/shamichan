// TODO: Generate and pass feed state on client init
// TODO: Cache init state
// TODO: Dispatch updates every 100 ms
// TODO: Grab clients needing init from registry on pulse

#[derive(Default)]
pub struct Pulsar {
	init_msg_cache: Vec<u8>,
}

super::gen_global_rwlock!(Pulsar);
