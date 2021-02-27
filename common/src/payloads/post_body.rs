use serde::{Deserialize, Serialize};
use std::hash::Hash;

// We opt to store strings as String even at the overhead of needing to convert
// back nad forth to Vec<char> for multibyte unicode support because it reduces
// memory usage almost 4 times. These will be stored in memory extensively on
// the server and client.

/// Node of the post body tree
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub enum Node {
	/// No content
	Empty,

	/// Start a new line
	NewLine,

	/// Contains a node and its next sibling. Allows building Node binary trees.
	//
	/// Using a binary tree structure instead of vectors of nodes allows writing
	/// a cleaner multithreaded parser and differ with less for loops with
	/// complicated exit conditions.
	Siblings([Box<Node>; 2]),

	/// Contains unformatted text. Can include newlines.
	Text(String),

	/// Link to another post
	PostLink(PostLink),

	/// Hash command result
	Command(Command),

	/// External URL
	URL(String),

	/// Configured reference to URL
	Reference { label: String, url: String },

	/// Link to embedadble resource
	Embed(Embed),

	/// Programming code tags
	Code(String),

	/// Spoiler tags
	//
	// TODO: make spoilers the top level tag (after code) to enable revealing
	/// it all on click or hover
	Spoiler(Box<Node>),

	/// Bold formatting tags
	Bold(Box<Node>),

	/// Italic formatting tags
	Italic(Box<Node>),

	/// Quoted Node list. Results from line starting with `>`.
	Quoted(Box<Node>),

	/// Node dependant on some database access or processing and pending
	/// finalization.
	/// Used by the server. These must never make it to the client.
	Pending(PendingNode),
}

impl Default for Node {
	#[inline]
	fn default() -> Self {
		Self::Empty
	}
}

impl Node {
	/// Diff the new post body against the old
	//
	// TODO: unit tests
	pub fn diff(&self, new: &Self) -> Option<PatchNode> {
		use Node::*;

		match (self, new) {
			(Empty, Empty) | (NewLine, NewLine) => None,
			(Siblings(old), Siblings(new)) => {
				macro_rules! diff {
					($i:expr) => {
						old[$i].diff(&*new[$i])
					};
				}

				match (diff!(0), diff!(1)) {
					(None, None) => None,
					(l @ _, r @ _) => Some(PatchNode::Siblings([
						l.map(|p| p.into()),
						r.map(|p| p.into()),
					])),
				}
			}
			(Text(old), Text(new))
			| (URL(old), URL(new))
			| (Code(old), Code(new)) => {
				// Hot path - most strings won't change and this will compare by
				// length first anyway
				if old == new {
					None
				} else {
					Some(PatchNode::Patch(TextPatch::new(
						&old.chars().collect::<Vec<char>>(),
						&new.chars().collect::<Vec<char>>(),
					)))
				}
			}
			(Spoiler(old), Spoiler(new))
			| (Bold(old), Bold(new))
			| (Italic(old), Italic(new))
			| (Quoted(old), Quoted(new)) => Self::diff(old, new),
			(old @ _, new @ _) => {
				if old != new {
					Some(PatchNode::Replace(new.clone()))
				} else {
					None
				}
			}
		}
	}

	/// Apply a patch AST to an existing post body AST
	//
	// TODO: unit tests
	pub fn patch(&mut self, patch: PatchNode) -> Result<(), String> {
		match (self, patch) {
			(dst @ _, PatchNode::Replace(p)) => {
				*dst = p;
			}
			(
				Node::Siblings([dst_l @ _, dst_r @ _]),
				PatchNode::Siblings([p_l, p_r]),
			) => {
				macro_rules! patch {
					($dst:expr, $p:expr) => {
						if let Some(p) = $p {
							$dst.patch(*p)?;
						}
					};
				}
				patch!(dst_r, p_r);
				patch!(dst_l, p_l);
			}
			(Node::Text(dst), PatchNode::Patch(p))
			| (Node::URL(dst), PatchNode::Patch(p))
			| (Node::Code(dst), PatchNode::Patch(p)) => {
				let mut new =
					String::with_capacity(p.estimate_new_size(dst.len()));
				p.apply(&mut new, dst.chars());
				*dst = new;
			}
			(dst @ _, p @ _) => {
				return Err(format!(
					"node type mismatch: attempting to patch {:#?}\nwith {:#?}",
					dst, p
				));
			}
		};
		Ok(())
	}
}

/// Node dependant on some database access or processing and pending
/// finalization.
/// Used by the server. These must never make it to the client.
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum PendingNode {
	Flip,
	EightBall,
	Pyu,
	PCount,

	/// Seconds to count down
	Countdown(u64),

	/// Hours to ban self for
	Autobahn(u16),

	Dice {
		/// Amount to offset the sum of all throws by
		offset: i16,

		/// Faces of the die
		faces: u16,

		/// Rolls to perform
		rolls: u8,
	},

	/// Pending post location fetch from the DB
	PostLink(u64),
}

/// Link to another post
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub struct PostLink {
	pub id: u64,

	/// If thread = 0, link has not had it's parenthood looked up yet on the
	/// server
	pub thread: u64,

	pub page: u32,
}

/// Hash command result
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum Command {
	/// Describes the parameters and results of one dice throw
	Dice {
		/// Amount to offset the sum of all throws by
		offset: i16,

		/// Faces of the die
		faces: u16,

		/// Results of dice throws. One per throw.
		results: Vec<u16>,
	},

	/// Coin flip
	Flip(bool),

	/// #8ball random answer dispenser
	EightBall(String),

	/// Synchronized countdown timer
	Countdown {
		start: u32,
		/// Unix timestamp
		secs: u32,
	},

	/// Self ban for N hours
	Autobahn(u16),

	/// Don't ask
	Pyu(u64),

	/// Don't ask
	PCount(u64),
}

/// Embedded content providers
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Hash, Copy)]
#[serde(rename_all = "snake_case")]
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

/// Describes and identifies a specific embedadble resource
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq, Hash)]
pub struct Embed {
	pub provider: EmbedProvider,
	pub data: String,
}

/// Patch to apply to an existing node
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum PatchNode {
	/// Replace node with new one
	Replace(Node),

	/// Descend deeper to patch one or both of the siblings
	Siblings([Option<Box<PatchNode>>; 2]),

	/// Partially modify an existing textual Node
	Patch(TextPatch),
}

/// Patch to apply to the text body of a post
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PostBodyPatch {
	pub id: u64,
	pub patch: PatchNode,
}

/// Partially modify an existing string
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub struct TextPatch {
	/// Position to start the mutation at
	pub position: u16,

	/// Number of characters to remove after position
	pub remove: u16,

	/// Text to insert at position after removal
	pub insert: Vec<char>,
}

impl TextPatch {
	/// Generate a patch from 2 versions of a string split into chars for
	/// multibyte unicode compatibility
	pub fn new(old: &[char], new: &[char]) -> Self {
		/// Find the first differing character in 2 character iterators
		#[inline]
		fn diff_i<'a, 'b>(
			a: impl Iterator<Item = &'a char>,
			b: impl Iterator<Item = &'b char>,
		) -> usize {
			a.zip(b).take_while(|(a, b)| a == b).count()
		}

		let start = diff_i(old.iter(), new.iter());
		let end = diff_i(old[start..].iter().rev(), new[start..].iter().rev());
		Self {
			position: start as u16,
			remove: (old.len() - end - start) as u16,
			insert: new[start..new.len() - end].iter().copied().collect(),
		}
	}

	/// Apply text patch to an existing string
	pub fn apply(
		&self,
		dst: &mut impl Extend<char>,
		mut src: impl Iterator<Item = char>,
	) {
		for _ in 0..self.position {
			dst.extend(src.next());
		}
		dst.extend(self.insert.iter().copied());
		dst.extend(src.skip(self.remove as usize));
	}

	/// Estimate size of destination after patch, assuming all characters are
	// single byte - true more often than not
	pub fn estimate_new_size(&self, dst_size: usize) -> usize {
		let mut s = dst_size as i16;
		s -= self.remove as i16;
		s += self.insert.len() as i16;

		// Protect against client-side attacks
		match s {
			0..=2000 => s as usize,
			_ => dst_size,
		}
	}
}

#[cfg(test)]
mod test {
	use super::TextPatch;

	// Test diffing and patching nodes
	#[test]
	fn node_diff() {
		// TODO
	}

	// Test diffing and patching text
	macro_rules! test_text_diff {
		($(
			$name:ident(
				$in:literal
				($pos:literal $remove:literal $insert:literal)
				$out:literal
			)
		)+) => {
			$(
				#[test]
				fn $name() {
					let std_patch = TextPatch{
						position: $pos,
						remove: $remove,
						insert: $insert.chars().collect(),
					};

					macro_rules! to_chars {
						($src:literal) => {{
							&$src.chars().collect::<Vec<char>>()
						}};
					}
					assert_eq!(
						TextPatch::new(to_chars!($in), to_chars!($out)),
						std_patch,
					);

					let mut res = String::new();
					std_patch.apply(&mut res, $in.chars());
					assert_eq!(res.as_str(), $out);
				}
			)+
		};
	}

	test_text_diff! {
		append(
			"a"
			(1 0 "a")
			"aa"
		)
		prepend(
			"bc"
			(0 0 "a")
			"abc"
		)
		append_to_empty_body(
			""
			(0 0 "abc")
			"abc"
		)
		backspace(
			"abc"
			(2 1 "")
			"ab"
		)
		remove_one_from_front(
			"abc"
			(0 1 "")
			"bc"
		)
		remove_one_multibyte_char(
			"αΒΓΔ"
			(2 1 "")
			"αΒΔ"
		)
		inject_into_the_middle(
			"abc"
			(2 0 "abc")
			"ababcc"
		)
		inject_multibyte_into_the_middle(
			"αΒΓ"
			(2 0 "Δ")
			"αΒΔΓ"
		)
		replace_in_the_middle(
			"abc"
			(1 1 "d")
			"adc"
		)
		replace_multibyte_in_the_middle(
			"αΒΓ"
			(1 1 "Δ")
			"αΔΓ"
		)
		replace_suffix(
			"abc"
			(1 2 "de")
			"ade"
		)
		replace_prefix(
			"abc"
			(0 2 "de")
			"dec"
		)
	}
}
