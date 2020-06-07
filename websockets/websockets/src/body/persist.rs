use protocol::{gen_global, payloads::post_body::Node};
use std::{collections::HashMap, sync::Arc};

gen_global! {, , HashMap<u64, Arc<Node>>}

// Buffer open post body changes and persist to DB once a second
pub fn persist_open_body(id: u64, body: Arc<Node>) {
	write(|m| m.insert(id, body));
}

// Periodically flush open bodies to DB
pub async fn flush_open_bodies() {
	loop {
		tokio::time::delay_for(std::time::Duration::from_secs(1)).await;

		// Don't keep mutex locked for DB writes. Swap the map with a fresh one
		// instead and work with that to reduce lock contention.
		let bodies = write(|m| std::mem::take(m));
		if bodies.is_empty() {
			continue;
		}
		if let Err(err) = crate::db::write_open_post_bodies(bodies).await {
			crate::bindings::log_error(&format!(
				"could not flush open posts bodies: {}",
				err
			));
		}
	}
}
