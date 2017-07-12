use super::{append, new_id, remove, set_inner_html, set_outer_html};
use std::collections::BTreeMap;
use std::fmt::Write;

// Element attributes
pub type Attrs = BTreeMap<String, Option<String>>;

// Represents an HTML Element
#[derive(Clone)]
pub struct Element {
	// ID of the element
	// If empty, an automatically generated ID will be assigned on first render.
	pub id: String,

	// HTML tag of the element
	pub tag: String,

	// Element attributes. Must not contain "id".
	// Omitting the value, will produce an attribute with no value.
	pub attrs: Attrs,

	// Inner HTML contents of Node. If Some, children are ignored.
	pub inner_html: Option<String>,

	// Child Elements
	pub children: Vec<Element>,
}

impl Element {
	// Write the Element and its subtree as HTML
	pub fn render(&mut self, w: &mut String) {
		if self.id == "" {
			self.id = new_id();
		}

		write!(w, "<{} id=\"{}\"", self.tag, &self.id).unwrap();
		for (ref key, val) in self.attrs.iter() {
			write!(w, " {}", key).unwrap();
			if let &Some(ref val) = val {
				write!(w, "=\"{}\"", &val).unwrap();
			}
		}
		w.push('>');

		if let Some(ref html) = self.inner_html {
			w.push_str(html);
		} else {
			for ch in self.children.iter_mut() {
				ch.render(w);
			}
		}

		write!(w, "</{}>", self.tag).unwrap();
	}

	// Diff Element against a subtree created on the last render.
	// When function returns, old will equal self.
	pub fn diff(&mut self, old: &mut Element) {
		// Completely replace node and subtree
		if self.id != old.id || self.tag != old.tag {
			let mut w = String::with_capacity(1 << 10);
			self.render(&mut w);
			set_outer_html(&old.id, &w);
			*old = self.clone();
			return;
		}

		self.diff_attrs(&mut old.attrs);

		// Account for all 4 possible transitions of inner_html
		if let Some(ref html) = self.inner_html {
			let mut same = false;
			if let Some(ref old) = old.inner_html {
				same = *html == *old;
			}
			if !same {
				set_inner_html(&self.id, html);
				old.inner_html = Some(html.clone());
				old.children.truncate(0);
			}
		} else {
			if let Some(_) = old.inner_html {
				set_inner_html(&self.id, "");
				old.inner_html = None;
				old.children.truncate(0);
			}
			self.diff_children(&mut old.children);
		}
	}

	fn diff_attrs(&self, old: &mut Attrs) {
		if self.attrs == *old {
			return;
		}

		// TODO: Diff and apply new arguments to element

		*old = self.attrs.clone();
	}

	fn diff_children(&mut self, old_ch: &mut Vec<Element>) {
		let mut diff = (self.children.len() as i32) - (old_ch.len() as i32);

		// Remove Elements from the end
		while diff < 0 {
			remove(&old_ch.pop().unwrap().id);
			diff += 1;
		}

		for (ref mut ch, ref mut old) in
			self.children.iter_mut().zip(old_ch.iter_mut()) {
			ch.diff(old);
		}

		// Append Elements
		if diff > 0 {
			let mut w = String::with_capacity(1 << 10);
			for ch in self.children.iter_mut().skip(old_ch.len()) {
				w.truncate(0);
				old_ch.push(ch.clone());
				ch.render(&mut w);
				append(&self.id, &w);
			}
		}
	}
}
