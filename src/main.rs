#![allow(dead_code)]

extern crate libc;

mod externs;
#[macro_use]
mod view;

use std::cell::RefCell;
use std::rc::Rc;
use view::*;

fn main() {
	let child = Rc::new(RefCell::new(Child {
	                                     id: new_id(),
	                                     data: String::from("Hello world!"),
	                                 }));
	let root = Rc::new(RefCell::new(Root {
	                                    id: new_id(),
	                                    child,
	                                }));
	let tree = Tree::new("hover-overlay", root);
}

struct Root {
	id: String,
	child: Rc<RefCell<Child>>,
}

impl State for Root {}

impl View for Root {
	implement_id!();

	fn children(&self) -> Vec<Box<View>> {
		vec![Box::new(self.child.borrow().clone())]
	}
}

#[derive(Hash, Clone)]
struct Child {
	id: String,
	data: String,
}

impl View for Child {
	implement_id!();

	fn render_inner(&self, w: &mut String) {
		w.push_str(&self.data);
	}
}
