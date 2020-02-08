use super::util;
use protocol::*;
use serde::{Deserialize, Serialize};
use std::hash::Hash;
use std::str;
use yew::agent::{Agent, AgentLink, Context, HandlerId};

// Key used to store AuthKey in local storage
const AUTH_KEY: &str = "auth_key";

// Global state singleton
pub struct State {
	link: AgentLink<Self>,

	// Authentication key
	auth_key: AuthKey,

	// Currently subscribed to thread or 0  (global thread index)
	feed: u64,

	// Subscriber registry
	subscribers: DoubleSetMap<SubscriptionType, HandlerId>,
}

// Value to subscribe to
#[derive(Serialize, Deserialize, Eq, PartialEq, Hash, Clone)]
pub enum SubscriptionType {
	FeedID,
	AuthKey,
}

#[derive(Serialize, Deserialize)]
pub enum Request {
	// Subscribe to updates of a value type. Will get sent the current value
	// on the next pass after this call.
	Subscribe(SubscriptionType),
}

#[derive(Serialize, Deserialize)]
pub enum Response {
	FeedID(u64),
	AuthKey(AuthKey),
}

impl Agent for State {
	type Reach = Context;
	type Message = ();
	type Input = Request;
	type Output = Response;

	fn create(link: AgentLink<Self>) -> Self {
		Self {
			link: link,
			feed: util::window()
				.location()
				.hash()
				.unwrap()
				.parse()
				.unwrap_or(0),
			auth_key: {
				// Read saved or generate a new authentication key
				let ls = util::local_storage();
				match ls.get_item(AUTH_KEY).unwrap() {
					Some(v) => {
						let mut key = AuthKey::default();
						match base64::decode_config_slice(
							&v,
							base64::STANDARD,
							key.as_mut(),
						) {
							Ok(_) => key,
							_ => Self::create_auth_key(),
						}
					}
					None => Self::create_auth_key(),
				}
			},
			subscribers: DoubleSetMap::default(),
		}
	}

	fn update(&mut self, _: Self::Message) {}

	fn handle_input(&mut self, req: Self::Input, id: HandlerId) {
		match req {
			Request::Subscribe(t) => {
				self.subscribers.insert(t.clone(), id);
				self.link.respond(
					id,
					match t {
						SubscriptionType::FeedID => Response::FeedID(self.feed),
						SubscriptionType::AuthKey => {
							Response::AuthKey(self.auth_key.clone())
						}
					},
				);
			}
		}
	}

	fn disconnected(&mut self, id: HandlerId) {
		self.subscribers.remove_by_value(&id);
	}
}

impl State {
	fn create_auth_key() -> AuthKey {
		let mut key = AuthKey::default();
		util::window()
			.crypto()
			.unwrap()
			.get_random_values_with_u8_array(key.as_mut())
			.unwrap();

		let mut buf: [u8; 88] =
			unsafe { std::mem::MaybeUninit::uninit().assume_init() };
		base64::encode_config_slice(key.as_mut(), base64::STANDARD, &mut buf);

		util::local_storage()
			.set_item(AUTH_KEY, unsafe { str::from_utf8_unchecked(&buf) })
			.unwrap();

		key
	}
}
