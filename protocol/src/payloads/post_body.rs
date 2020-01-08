use serde::{Deserialize, Serialize};

// Node of the post body tree
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(tag = "type", content = "content")]
pub enum Node {
	// Contains unformatted text
	Text(String),

	// Link to another post
	PostLink(PostLink),

	// Hash command result
	Command(Command),

	// External URL
	URL(String),

	// Configured reference to URL
	Reference { label: String, url: String },

	// URL to embedadble resource
	Embed(String),

	// Quoted Node list. Results from line starting with `>`.
	Quoted(Vec<Node>),

	// Spoiler tags
	Spoiler(Vec<Node>),

	// Programming code tags
	Code(Vec<Node>),

	// Bold formatting tags
	Bold(Vec<Node>),

	// Italic formatting tags
	Italic(Vec<Node>),

	// Red text formatting tags
	Red(Vec<Node>),

	// Blue text formatting tags
	Blue(Vec<Node>),
}

impl Default for Node {
	fn default() -> Self {
		Node::Text(Default::default())
	}
}

macro_rules! variant {
    ($name:ident {$($field:ident: $t:ty,)*}) => {
        #[derive(Serialize, Deserialize, Debug, Clone)]
        pub struct $name {
            $(pub $field: $t),*
        }
    }
}

// Link to another post
variant! { PostLink {
	id: u64,
	thread: u64,
}}

// Hash command result
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(tag = "type", content = "content")]
pub enum Command {
	// Describes the parameters and results of one dice throw
	Dice {
		// Amount to offset the sum of all throws by
		offset: i16,

		// Faces of the die
		faces: u16,

		// Results of dice throws. One per throw.
		results: Vec<u16>,
	},

	// Coin flip
	Flip(bool),

	// #8ball random answer dispenser
	EightBall(String),

	// Synchronized timer command type for synchronizing episode time during
	// group anime watching and such
	SyncWatch {
		// Requested time duration in seconds
		duration: u64,

		// Timer start Unix timestamp
		start: u64,

		// Timer end Unix timestamp
		end: u64,
	},

	// Don't ask
	Pyu(u64),

	// Don't ask
	Pcount(u64),

	// Self ban for N hours
	AutoBahn(u16),
}
