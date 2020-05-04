use serde::{Deserialize, Serialize};

// Node of the post body tree
#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(tag = "type", content = "content")]
pub enum Node {
	// Contains unformatted text. Can include newlines.
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

	// Programming code tags
	//
	// TODO: Run code highlighting using syntect server-side
	// TODO: Allow `` to be followed by a supported language. Example: ``rust.
	// TODO: Fallback to plaintext
	Code(String),

	// Spoiler tags
	//
	// TODO: make spoilers the top level tag (after code) to enable revealing
	// it all on click or hover
	Spoiler(Vec<Node>),

	// Quoted Node list. Results from line starting with `>`.
	Quoted(Vec<Node>),

	// Bold formatting tags
	Bold(Vec<Node>),

	// Italic formatting tags
	Italic(Vec<Node>),
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
	page: u32,
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

	// Synchronized countdown timer
	Countdown {
		// Unix timestamps
		start: u32,
		end: u32,
	},

	// Self ban for N hours
	AutoBahn(u16),
}
