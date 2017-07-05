use externs::*;
use std::cell::RefCell;
use std::collections::BTreeMap;
use std::collections::HashSet;
use std::collections::hash_map::DefaultHasher;
use std::fmt::Write;
use std::hash::Hash;
use std::hash::Hasher;
use std::rc::Rc;

static mut ID_COUNTER: u64 = 0;

// Generate a new unique view ID
pub fn new_id() -> String {
	unsafe { ID_COUNTER += 1 };
	format!("brunhild-{}", unsafe { ID_COUNTER })
}

// Attributes of a view's root element
pub type Attributes = BTreeMap<String, Option<String>>;

// Hashes the state of the view. Used for diffing.
// For performance reasons the hash of a parent view should not reflect changes
// in its children. This will produce the same result, but is needlessly costly.
// Static views should use the default trait implementation.
// Non-parent views can implement State by simply deriving Hash.
pub trait State {
	fn state(&self) -> u64 {
		0
	}
}

// Enables views to implement State, by simply deriving Hash.
impl<H> State for H
    where H: Hash
{
	fn state(&self) -> u64 {
		let mut h = DefaultHasher::new();
		self.hash(&mut h);
		h.finish()
	}
}

// Base unit of manipulation. Set CH to type of child view, if the view will be
// able to have child views.
pub trait View<CH: View = NOOP>: State {
	// Return the ID of a view. All views must store a constant ID.
	// IDs chosen by the user must be unique.
	// If you do not wish to assign a custom ID, generate one with new_id().
	fn id(&self) -> String;

	// Returns the tag of the root element
	fn tag(&self) -> String {
		String::from("div")
	}

	// Returns attributes of the root element. Should not contain "id".
	fn attrs(&self) -> Attributes {
		BTreeMap::new()
	}

	// Renders the inner contents of the view. Should be left default for views
	// with child views.
	fn render_inner(&self, &mut String) {}

	// Returns child views. Should be left default for views, that implement
	// render_inner().
	fn children(&self) -> Vec<CH> {
		Vec::new()
	}
}

// This view does nothing. Acts as a child type for views without children.
#[allow(private_in_public)]
struct NOOP {
	id: String,
}

impl State for NOOP {}

impl<'a> View for NOOP {
	fn id(&self) -> String {
		String::new()
	}
}

pub struct Tree<T>
	where T: for<'a> View
{
	view: Rc<RefCell<T>>,
	node: Node,
	updated: HashSet<String>,
}

impl<T> Tree<T>
    where T: for<'a> View
{
	pub fn new(parent_id: &str, v: Rc<RefCell<T>>) -> Tree<T> {
		// TODO: Register render function with RAF

		let mut w = String::with_capacity(1 << 10);
		let node = Node::new(&*v.borrow(), &mut w);
		append_element(parent_id, &mut w);
		Tree {
			view: v.clone(),
			node,
			updated: HashSet::new(),
		}
	}

	fn diff(&mut self) {
		self.node
			.check_marked(&mut self.updated, &*self.view.borrow());
		self.updated.clear();
	}

	// Mark view and its children as updated and thus needing a diff.
	pub fn update<V: View>(&mut self, v: V) {
		self.updated.insert(v.id());
	}
}

struct Node {
	tag: String,
	id: String,
	value: String, // Will be used for storing state of input elements
	state: u64,
	attrs: Attributes,
	children: Vec<Node>,
}

impl Node {
	fn new<V: View>(v: &V, w: &mut String) -> Node {
		let id = v.id();
		let tag = v.tag();
		let attrs = v.attrs();

		// Render element
		write!(w, "<{} id=\"{}\"", tag, id).unwrap();
		for (ref key, val) in attrs.iter() {
			write!(w, " {}", key).unwrap();
			if let &Some(ref val) = val {
				write!(w, "=\"{}\"", &val).unwrap();
			}
		}
		w.push('>');
		v.render_inner(w);
		let children = v.children().iter().map(|v| Node::new(v, w)).collect();
		write!(w, "</{}>", tag).unwrap();

		Node {
			id,
			tag: tag,
			attrs,
			state: v.state(),
			children,
			value: String::new(),
		}
	}

	// Check, if node is marked as updated.
	fn check_marked<V: View>(&mut self, marked: &mut HashSet<String>, v: &V) {
		// Diff the node and its subtree
		if marked.contains(&self.id) {
			marked.remove(&self.id);
			self.diff(v);
			return;
		}

		// Descend down the subtree, checking for marked nodes
		for (ref mut n, v) in
			self.children.iter_mut().zip(v.children().iter()) {
			n.check_marked(marked, v);
		}
	}

	fn diff<V: View>(&mut self, v: &V) {
		// Completely replace node and subtree
		if v.id() != self.id {
			let old_ID = self.id.clone();
			let mut w = String::with_capacity(1 << 10);
			*self = Node::new(v, &mut w);
			return set_outer_HTML(&old_ID, &w);
		}

		let state = v.state();
		let mut changed = false;
		if state != self.state {
			self.state = state;
			changed = true;
			self.diff_attrs(v.attrs());
		}

		let children = v.children();
		if self.children.len() == 0 && children.len() == 0 {
			if changed {
				let mut w = String::with_capacity(1 << 10);
				v.render_inner(&mut w);
				return set_inner_HTML(&self.id, &w);
			}
		} else {
			self.diff_children(children);
		}
	}

	fn diff_attrs(&mut self, attrs: Attributes) {
		if self.attrs == attrs {
			return;
		}

		// TODO: Diff and apply new arguments to element

		self.attrs = attrs;
	}

	fn diff_children<V: View>(&mut self, views: Vec<V>) {
		let diff = (views.len() as i32) - (self.children.len() as i32);

		// Remove nodes from the end
		if diff < 0 {
			pop_children(&self.id, -diff);
			self.children.truncate(views.len())
		}

		for (ref mut n, v) in self.children.iter_mut().zip(views.iter()) {
			n.diff(v);
		}

		// Append nodes
		if diff > 0 {
			let mut w = String::with_capacity(1 << 10);
			for ch in views.iter().skip(self.children.len()) {
				w.truncate(0);
				self.children.push(Node::new(ch, &mut w));
				append_element(&self.id, &w);
			}
		}
	}
}
