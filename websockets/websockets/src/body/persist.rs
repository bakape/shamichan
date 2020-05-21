use protocol::{gen_global, payloads::post_body::Node};
use std::{collections::HashMap, sync::Arc};

gen_global! {, , HashMap<u64, Arc<Node>>}

// Buffer open post body changes and persist to DB once a second
pub fn persist_open_body(id: u64, body: Arc<Node>) {
	use std::sync::Once;

	static ONCE: Once = Once::new();
	ONCE.call_once(|| {
		std::thread::Builder::new()
			.name("open_body_flusher".into())
			.spawn(flush_open_bodies)
			.unwrap();
	});

	write(|m| m.insert(id, body));
}

fn flush_open_bodies() {
	loop {
		std::thread::sleep(std::time::Duration::from_secs(1));

		// Don't keep mutex locked for DB writes. Swap the map with a fresh one
		// instead and work with that to reduce lock contention.
		let bodies = write(|m| std::mem::take(m));
		if bodies.is_empty() {
			continue;
		}
		if let Err(err) = move || -> crate::common::DynResult {
			crate::bindings::write_open_post_bodies(&serde_json::to_vec(
				&bodies,
			)?)?;
			Ok(())
		}() {
			crate::bindings::log_error(&format!(
				"could not flush open posts bodies: {}",
				err
			));
		}
	}
}
