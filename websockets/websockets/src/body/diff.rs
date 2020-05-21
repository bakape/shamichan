use protocol::payloads::post_body::{Node, PatchNode};

// Diff the new post body against the old
pub fn diff(old: &Node, new: &Node) -> Option<PatchNode> {
	use Node::*;

	match (old, new) {
		(Empty, Empty) | (NewLine, NewLine) => None,
		(Siblings(old), Siblings(new)) => {
			macro_rules! diff {
				($i:expr) => {
					|| diff(&*old[$i], &*new[$i])
				};
			}

			match rayon::join(diff!(0), diff!(1)) {
				(None, None) => None,
				(l @ _, r @ _) => Some(PatchNode::Siblings([
					l.map(|p| p.into()),
					r.map(|p| p.into()),
				])),
			}
		}
		(Text(new), Text(old))
		| (URL(old), URL(new))
		| (Code(old), Code(new)) => diff_text(old, new),
		(Spoiler(old), Spoiler(new))
		| (Bold(old), Bold(new))
		| (Italic(old), Italic(new))
		| (Quoted(old), Quoted(new)) => diff(old, new),
		(old @ _, new @ _) => {
			if old != new {
				Some(PatchNode::Replace(new.clone()))
			} else {
				None
			}
		}
	}
}

// Diff text and return patching instructions to enable at least some
// differential compression for string updates
fn diff_text(old: &str, new: &str) -> Option<PatchNode> {
	// Hot path - most strings won't change and this will compare by length
	// first anyway
	if old == new {
		return None;
	}

	// Split into chars for multibyte unicode compatibility
	let old_r = old.chars().collect::<Vec<char>>();
	let new_r = new.chars().collect::<Vec<char>>();

	// Find the first differing character in 2 character iterators
	fn diff_i<'a, 'b>(
		mut a: impl Iterator<Item = &'a char>,
		mut b: impl Iterator<Item = &'b char>,
	) -> usize {
		let mut i = 0;
		loop {
			if a.next() != b.next() {
				return i;
			}
			i += 1;
		}
	}

	let start = diff_i(old_r.iter(), new_r.iter());
	let end = diff_i(old_r[start..].iter().rev(), new_r[start..].iter());

	Some(PatchNode::Patch(protocol::payloads::post_body::TextPatch {
		position: start as u16,
		remove: (old_r.len() - end - start) as u16,
		insert: new_r[start..new_r.len() - end].into_iter().collect(),
	}))
}
