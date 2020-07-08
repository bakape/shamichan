use serde::{Deserialize, Serialize};
use std::hash::Hash;

// Node of the post body tree
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub enum Node {
	// No content
	Empty,

	// Start a new line
	NewLine,

	// Contains a node and its next sibling. Allows building Node binary trees.
	//
	// Using a binary tree structure instead of vectors of nodes allows writing
	// a cleaner multithreaded parser and differ with less for loops with
	// complicated exit conditions.
	Siblings([Box<Node>; 2]),

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

	// Link to embedadble resource
	Embed(Embed),

	// Programming code tags
	Code(String),

	// Spoiler tags
	//
	// TODO: make spoilers the top level tag (after code) to enable revealing
	// it all on click or hover
	Spoiler(Box<Node>),

	// Bold formatting tags
	Bold(Box<Node>),

	// Italic formatting tags
	Italic(Box<Node>),

	// Quoted Node list. Results from line starting with `>`.
	Quoted(Box<Node>),

	// Node dependant on some database access or processing and pending
	// finalization.
	// Used by the server. These must never make it to the client.
	Pending(PendingNode),
}

impl Default for Node {
	fn default() -> Self {
		Self::Empty
	}
}

// Node dependant on some database access or processing and pending
// finalization.
// Used by the server. These must never make it to the client.
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub enum PendingNode {
	Flip,
	EightBall,
	Pyu,
	PCount,

	// Seconds to count down
	Countdown(u64),

	// Hours to ban self for
	Autobahn(u16),

	Dice {
		// Amount to offset the sum of all throws by
		offset: i16,

		// Faces of the die
		faces: u16,

		// Rolls to perform
		rolls: u8,
	},
}

// Link to another post
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub struct PostLink {
	pub id: u64,

	// If thread = 0, link has not had it's parenthood looked up yet on the
	// server
	pub thread: u64,

	pub page: u32,
}

// Hash command result
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
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
		start: u32, // Unix timestamp
		secs: u32,
	},

	// Self ban for N hours
	Autobahn(u16),

	// Don't ask
	Pyu(u64),

	// Don't ask
	PCount(u64),
}

// Embedded content providers
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Hash, Copy)]
pub enum EmbedProvider {
	YouTube,
	SoundCloud,
	Vimeo,
	Coub,
	Twitter,
	Imgur,
	BitChute,
	Invidious,
}

// Describes and identifies a specific embedadble resource
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq, Hash)]
pub struct Embed {
	pub provider: EmbedProvider,
	pub data: String,
}

// Patch to apply to an existing node
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub enum PatchNode {
	// Replace node with new one
	Replace(Node),

	// Descend deeper to patch one or both of the siblings
	Siblings([Option<Box<PatchNode>>; 2]),

	// Partially modify an existing Code, Text or URL Node
	Patch(TextPatch),
}

// Partially modify an existing string
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub struct TextPatch {
	// Position to start the mutation at
	pub position: u16,

	// Number of characters to remove after position
	pub remove: u16,

	// Text to insert at position after removal
	pub insert: Vec<char>,
}

impl TextPatch {
	// Generate a patch from 2 versions of a string split into chars for
	// multibyte unicode compatibility
	pub fn new(old: &[char], new: &[char]) -> Self {
		// Find the first differing character in 2 character iterators
		fn diff_i<'a, 'b>(
			mut a: impl Iterator<Item = &'a char>,
			mut b: impl Iterator<Item = &'b char>,
		) -> usize {
			let mut i = 0;
			while a.next() == b.next() {
				i += 1;
			}
			return i;
		}

		let start = diff_i(old.iter(), new.iter());
		let end = diff_i(old[start..].iter().rev(), new[start..].iter());
		Self {
			position: start as u16,
			remove: (old.len() - end - start) as u16,
			insert: new[start..new.len() - end].iter().copied().collect(),
		}
	}
}
