use std::cell::RefCell;
use std::collections::BTreeMap;
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

// Should not contain "id"
pub type Attributes = BTreeMap<String, Option<String>>;

pub trait State {
	fn state(&self) -> u64 {
		0
	}
}

impl<H> State for H
    where H: Hash
{
	fn state(&self) -> u64 {
		let mut h = DefaultHasher::new();
		self.hash(&mut h);
		h.finish()
	}
}

// Base unit of manipulation
pub trait View<'a, CH: View<'a> = NOOP>: State {
	fn tag(&self) -> &'a str {
		"div"
	}

	fn id(&self) -> Option<&'a str> {
		None
	}

	fn attrs(&self) -> Attributes {
		BTreeMap::new()
	}

	fn children(&self) -> Vec<Rc<RefCell<CH>>> {
		Vec::new()
	}

	fn render(&self, w: &mut String) -> Node {
		let id = match self.id() {
			Some(id) => String::from(id),
			None => new_id(),
		};
		let tag = self.tag();
		let attrs = self.attrs();

		// Render element
		write!(w, "<{} id=\"{}\"", tag, &id).unwrap();
		for (ref key, val) in attrs.iter() {
			write!(w, " {}", key).unwrap();
			if let &Some(ref val) = val {
				write!(w, "=\"{}\"", &val).unwrap();
			}
		}
		w.push('>');
		self.render_inner(w);
		let children = self.children()
			.iter()
			.map(|v| v.borrow().render(w))
			.collect();
		write!(w, "</{}>", tag).unwrap();

		Node {
			id,
			tag: String::from(tag),
			attrs,
			state: self.state(),
			children,
		}
	}

	fn render_inner(&self, &mut String) {}
}

pub struct NOOP;

impl State for NOOP {}

impl<'a> View<'a> for NOOP {}

pub struct Tree<T>
	where T: for<'a> View<'a>
{
	view: Rc<RefCell<T>>,
	node: Node,
}

impl<T> Tree<T>
    where T: for<'a> View<'a>
{
	pub fn new(parent_id: &str, v: Rc<RefCell<T>>) -> Tree<T> {
		// TODO: Insert into DOM
		// TODO: Register render function with RAF

		let mut w = String::with_capacity(1 << 10);
		let node = v.borrow().render(&mut w);
		Tree {
			view: v.clone(),
			node,
		}
	}

	fn diff(&mut self) {
		self.node.diff(&*self.view.borrow())
	}
}

struct Node {
	pub tag: String,
	pub id: String,
	pub state: u64,
	pub attrs: Attributes,
	pub children: Vec<Node>,
}

impl Node {
	fn diff<'a, V: View<'a>>(&mut self, v: &V) {
		if v.tag() != self.tag {
			return self.replace(v);
		}
		if let Some(id) = v.id() {
			if id != self.id {
				return self.replace(v);
			}
		}

		let children = v.children();
		if self.children.len() == 0 && children.len() == 0 {
			if v.state() != self.state {
				self.diff_attrs(v.attrs());
				let mut w = String::with_capacity(1 << 10);
				v.render(&mut w);
				// TODO: Replace contents
			}
		} else {
			if v.state() != self.state {
				self.diff_attrs(v.attrs());
			}
			self.diff_children(children);
		}
	}

	fn replace<'a, V: View<'a>>(&mut self, v: &V) {
		let old_ID = self.id.clone();
		let mut w = String::with_capacity(1 << 10);
		*self = v.render(&mut w);
		// TODO: Replace element
	}

	fn diff_attrs(&mut self, attrs: Attributes) {
		if self.attrs == attrs {
			return;
		}

		// TODO: Diff and apply new arguments to element

		for (key, _) in &attrs {
			assert!(key == "id", "attribute has 'id' key");
		}
		self.attrs = attrs;
	}

	fn diff_children<'a, V: View<'a>>(&mut self, views: Vec<Rc<RefCell<V>>>) {
		let diff = (views.len() as i32) - (self.children.len() as i32);
		if diff > 0 {
			// TODO: Append elements
		} else if diff < 0 {
			// TODO: Remove elements
		}

		for (ref mut n, v) in self.children.iter_mut().zip(views.iter()) {
			n.diff(&*v.borrow());
		}
	}
}
