use serde::{Deserialize, Serialize};
use std::{hash::Hash, ops::AddAssign};

// We opt to store strings as String even at the overhead of needing to convert
// back nad forth to Vec<char> for multibyte unicode support because it reduces
// memory usage almost 4 times. These will be stored in memory extensively on
// the server and client.

/// Node of the post body tree
//
// TODO: bump allocation for entire tree to reduce allocation/deallocation
// overhead. Depends on https://github.com/rust-lang/rust/issues/32838
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
pub enum Node {
	/// No content
	Empty,

	/// Start a new line
	Newline,

	/// Contains a list of child nodes.
	///
	/// A list with a single Node must be handled just like that singe Node.
	Children(Vec<Node>),

	/// Contains unformatted text. Can include newlines.
	Text(String),

	/// Link to another post
	PostLink {
		/// Post the link points to
		id: u64,

		/// Target post's parent thread
		///
		/// If thread = 0, link has not had it's parenthood looked up yet on the
		/// server
		thread: u64,

		/// Parent page of target post
		page: u32,
	},

	/// Hash command result
	Command(Command),

	/// External URL
	URL(String),

	/// Configured reference to URL
	Reference { label: String, url: String },

	/// Link to embedadble resource
	Embed {
		/// Provider of embedadble resource
		provider: EmbedProvider,

		/// Original URL matched by the server.
		///
		/// Persisting this instead of some parsed result is more flexible, as
		/// it allows switching embedding schemes easily in the future. The
		/// client can simply fallback to plain URLs in case of failure.
		url: String,
	},

	/// Programming code tags
	Code(String),

	/// Spoiler tags
	Spoiler(Box<Node>),

	/// Bold formatting tags
	Bold(Box<Node>),

	/// Italic formatting tags
	Italic(Box<Node>),

	/// Quoted Node list. Results from line starting with `>`.
	Quoted(Box<Node>),

	/// Node dependant on some database access or processing and pending
	/// finalization.
	Pending(PendingNode),
}

impl Default for Node {
	#[inline]
	fn default() -> Self {
		Self::Empty
	}
}

impl Node {
	/// Construct a new text node
	#[inline]
	pub fn text(s: impl Into<String>) -> Node {
		Node::Text(s.into())
	}

	/// Construct a new quoted node
	#[inline]
	pub fn quote(inner: Node) -> Node {
		Node::Quoted(inner.into())
	}

	/// Construct a new spoiler node
	#[inline]
	pub fn spoiler(inner: Node) -> Node {
		Node::Spoiler(inner.into())
	}

	/// Diff the new post body against the old
	pub fn diff(&self, new: &Self) -> Option<Patch> {
		use Node::*;

		match (self, new) {
			(Empty, Empty) | (Newline, Newline) => None,
			(Children(old), Children(new)) => {
				let mut patch = vec![];
				let mut truncate = None;
				let mut append = vec![];

				let mut old_it = old.iter();
				let mut new_it = new.iter();
				let mut i = 0;
				loop {
					match (old_it.next(), new_it.next()) {
						(Some(o), Some(n)) => {
							if let Some(p) = o.diff(n) {
								patch.push((i, p));
							}
						}
						(None, Some(n)) => {
							append.push(n.clone());
							append.extend(new_it.map(Clone::clone));
							break;
						}
						(Some(_), None) => {
							truncate = Some(i);
							break;
						}
						(None, None) => break,
					};
					i += 1;
				}

				if patch.is_empty() && truncate.is_none() && append.is_empty() {
					None
				} else {
					Some(Patch::Children {
						patch,
						truncate,
						append,
					})
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
					Some(Patch::Text(TextPatch::new(
						&old.chars().collect::<Vec<char>>(),
						&new.chars().collect::<Vec<char>>(),
					)))
				}
			}
			(Spoiler(old), Spoiler(new))
			| (Bold(old), Bold(new))
			| (Italic(old), Italic(new))
			| (Quoted(old), Quoted(new)) => {
				Self::diff(old, new).map(|p| Patch::Wrapped(p.into()))
			}
			(old @ _, new @ _) => {
				if old != new {
					Some(Patch::Replace(new.clone()))
				} else {
					None
				}
			}
		}
	}

	/// Apply a patch tree to a post body tree
	pub fn patch(&mut self, patch: Patch) -> Result<(), String> {
		Ok(match (self, patch) {
			(dst @ _, Patch::Replace(p)) => {
				*dst = p;
			}
			(
				Node::Children(dst),
				Patch::Children {
					patch,
					truncate,
					append,
				},
			) => {
				for (i, p) in patch {
					let l = dst.len();
					dst.get_mut(i)
						.ok_or_else(|| {
							format!("patch out of bounds: {} >= {}", i, l)
						})?
						.patch(p)?;
				}
				if let Some(len) = truncate {
					dst.truncate(len);
				}
				dst.extend(append);
			}

			(Node::Text(dst), Patch::Text(p))
			| (Node::URL(dst), Patch::Text(p))
			| (Node::Code(dst), Patch::Text(p)) => {
				let mut new =
					String::with_capacity(p.estimate_new_size(dst.len()));
				p.apply(&mut new, dst.chars());
				*dst = new;
			}
			(Node::Spoiler(old), Patch::Wrapped(p))
			| (Node::Bold(old), Patch::Wrapped(p))
			| (Node::Italic(old), Patch::Wrapped(p))
			| (Node::Quoted(old), Patch::Wrapped(p)) => {
				old.patch(*p)?;
			}
			(dst @ _, p @ _) => {
				return Err(format!(
					"node type mismatch: attempting to patch {:#?}\nwith {:#?}",
					dst, p
				));
			}
		})
	}
}

/// Extends an existing String as efficiently as possible.
/// Also supports being converted into a new String.
pub trait ExtendString: Sized {
	/// Extends an existing String as efficiently as possible
	fn extend_string(&self, dst: &mut String);

	/// Consumes value to construct a String
	fn into_string(self) -> String {
		let mut s = String::new();
		self.extend_string(&mut s);
		s
	}
}

impl ExtendString for String {
	fn extend_string(&self, dst: &mut String) {
		*dst += self;
	}

	// Avoids copy
	fn into_string(self) -> String {
		self
	}
}

impl ExtendString for &str {
	fn extend_string(&self, dst: &mut String) {
		*dst += self;
	}
}

impl ExtendString for char {
	fn extend_string(&self, dst: &mut String) {
		dst.push(*self);
	}
}

impl ExtendString for u8 {
	fn extend_string(&self, dst: &mut String) {
		dst.push(*self as char);
	}
}

impl AddAssign<Node> for Node {
	/// If pushing a Children to a Children, the destination list is extended.
	/// If pushing a Text to a Text, the destination Text is extended.
	/// Conversions from non-Children and Empty is automatically handled.
	fn add_assign(&mut self, rhs: Node) {
		use Node::*;

		match (self, rhs) {
			(_, Empty) => (),
			(dst @ Empty, n @ _) => *dst = n,
			// Merge adjacent strings
			(Text(s), Text(n)) => *s += &n,
			(Children(v), Children(n)) => {
				let mut it = n.into_iter();
				match (v.last_mut(), it.next()) {
					// Merge adjacent strings
					(Some(Text(dst)), Some(Text(s))) => *dst += &s,
					(_, Some(n @ _)) => v.push(n),
					_ => (),
				};
				v.extend(it);
			}
			(Children(v), Text(s)) => match v.last_mut() {
				// Merge adjacent strings
				Some(Text(dst)) => *dst += &s,
				_ => v.push(Text(s)),
			},
			(Children(v), n @ _) => v.push(n),
			(dst @ _, n @ _) => {
				*dst = Node::Children(vec![std::mem::take(dst), n])
			}
		};
	}
}

impl<T> AddAssign<T> for Node
where
	T: ExtendString,
{
	/// Avoids allocations in comparison to += Node.
	fn add_assign(&mut self, rhs: T) {
		use Node::*;

		match self {
			Text(dst) => rhs.extend_string(dst),
			Children(v) => match v.last_mut() {
				Some(Text(dst)) => rhs.extend_string(dst),
				_ => v.push(Text(rhs.into_string())),
			},
			_ => {
				*self += Node::Text(rhs.into_string());
			}
		};
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
	DropBox,
}

/// Patch to apply to an existing node
#[derive(Serialize, Deserialize, Debug, Clone, Eq, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum Patch {
	/// Replace node with new one
	Replace(Node),

	/// Partially modify an existing textual Node
	Text(TextPatch),

	/// Patch the contents of a wrapped Node like Spoiler, Quoted, Bold and
	/// Italic
	Wrapped(Box<Patch>),

	/// Descend deeper to patch children the specified order
	Children {
		/// First patch nodes at the specific indices
		patch: Vec<(usize, Patch)>,

		/// Then truncate child list to match this size
		truncate: Option<usize>,

		/// Then append these nodes
		append: Vec<Node>,
	},
}

/// Patch to apply to the text body of a post
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PostBodyPatch {
	pub id: u64,
	pub patch: Patch,
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

	mod nodes {
		use super::super::*;
		use paste::paste;
		use Node::*;

		/// Create a list of child nodes
		macro_rules! children {
		($($ch:expr),*$(,)?) => {
				Node::Children(vec![ $($ch,)* ])
			};
		}

		fn text(s: impl Into<String>) -> Node {
			Node::Text(s.into())
		}

		macro_rules! test_diff {
			($(
				$name:ident($in_out:expr)
			)+) => {
				$(
					#[test]
					fn $name() {
						let patch = $in_out.diff(&$in_out);
						if patch.is_some() {
							panic!("expected no patch, got: {:#?}", patch);
						}
					}
				)+
			};
			($(
				$name:ident(
					$in:expr
					=> $patch:expr
					=> $out:expr
				)
			)+) => {
				$(
					#[test]
					fn $name() -> Result<(), String> {
						let mut input = $in;
						let output = $out;
						let patch = input.diff(&output);

						macro_rules! assert_eq {
							($got:expr, $expected:expr) => {
								assert!(
									$got == $expected,
									"\ngot:      {:#?}\nexpected: {:#?}\n",
									$got,
									$expected,
								);
							};
						}

						assert_eq!(patch, Some($patch));

						input.patch(patch.unwrap())?;
						assert_eq!(input, output);

						Ok(())
					}
				)+
			};
		}

		macro_rules! test_text {
			($( $variant:ident )+) => {
				$(
					paste! {
						test_diff! {
							[<diff_inside_ $variant:lower>](
								$variant("a".into())
								=> Patch::Text(TextPatch{
									position: 1,
									remove: 0,
									insert: vec!['a'],
								})
								=> $variant("aa".into())
							)
						}
						test_diff! {
							[<identical_inside_ $variant:lower>](
								$variant("foo".into())
							)
						}
					}
				)+
			};
		}

		test_text! {
			Text
			URL
			Code
		}

		test_diff! {
			replace_node(
				text("foo")
				=> Patch::Replace(children![
					text("foo"),
					text("bar"),
				])
				=> children![
					text("foo"),
					text("bar"),
				]
			)
			append(
				children![
					text("foo"),
					Newline,
				]
				=> Patch::Children{
					patch: vec![],
					truncate: None,
					append: vec![text("bar")],
				}
				=> children![
					text("foo"),
					Newline,
					text("bar"),
				]
			)
			patch_child(
				children![
					text("foo"),
					Newline,
				]
				=> Patch::Children{
					patch: vec![
						(
							0,
							Patch::Text(
								TextPatch {
									position: 3,
									remove: 0,
									insert: vec!['l'],
								},
							),
						),
					],
					truncate: None,
					append: vec![],
				}
				=> children![
					text("fool"),
					Newline,
				]
			)
			append_and_patch(
				children![
					text("foo"),
					Newline,
				]
				=> Patch::Children{
					patch: vec![
						(
							0,
							Patch::Text(

								TextPatch {
									position: 3,
									remove: 0,
									insert: vec!['l'],
								},
							),
						),
					],
					truncate: None,
					append: vec![text("kono")],
				}
				=> children![
					text("fool"),
					Newline,
					text("kono"),
				]
			)
			truncate_and_patch(
				children![
					text("foo"),
					Newline,
					text("kono"),
				]
				=> Patch::Children{
					patch: vec![
						(
							0,
							Patch::Text(
								TextPatch {
									position: 3,
									remove: 0,
									insert: vec!['l'],
								},
							),
						),
					],
					truncate: Some(2),
					append: vec![],
				}
				=> children![
					text("fool"),
					Newline,
				]
			)
			multiple_levels_of_children(
				children![
					text("kono"),
					children![
						text("foo"),
					],
				]
				=> Patch::Children{
					patch: vec![
						(
							0,
							Patch::Text(
								TextPatch {
									position: 4,
									remove: 0,
									insert: "suba".chars().collect(),
								},
							),
						),
						(
							1,
							Patch::Children{
								patch: vec![
									(
										0,
										Patch::Text(
											TextPatch{
												position: 3,
												remove: 0,
												insert: vec!['l'],
											},
										),
									),
								],
								truncate: None,
								append: vec![],
							},
						),
					],
					truncate: None,
					append: vec![Newline],
				}
				=> children![
					text("konosuba"),
					children![
						text("fool"),
					],
					Newline,
				]
			)
		}

		test_diff! {
			empty(Empty)
			newline(Newline)
			identical_text(text("foo"))
			identical_pending(Pending(PendingNode::Flip))
			empty_child_list(children![])
			identical_children(children![text("foo")])
			identical_nested_children(children![
				children![
					text("foo"),
				],
				text("bar"),
			])
		}

		macro_rules! test_inside_formatting {
			($( $tag:ident )+) => {
				$(
					paste! {
						test_diff! {
							[<diff_inside_ $tag:lower>](
								$tag(text("a").into())
								=> Patch::Wrapped(
									Patch::Text(TextPatch{
										position: 1,
										remove: 0,
										insert: vec!['a'],
									})
									.into()
								)
								=> $tag(text("aa").into())
							)
						}
						test_diff! {
							[<identical_inside_ $tag:lower>](
								$tag(text("foo").into())
							)
						}
					}
				)+
			};
		}

		test_inside_formatting! {
			Spoiler
			Bold
			Italic
			Quoted
		}
	}

	mod text {
		use super::super::*;

		// Test diffing and patching text
		macro_rules! test_text_diff_patch {
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

		test_text_diff_patch! {
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
}
