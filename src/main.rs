extern crate libc;

mod externs;
#[macro_use]
mod view;

use std::cell::RefCell;
use std::fmt::Write;
use std::rc::Rc;
use view::*;

fn main() {
	let child = Rc::new(RefCell::new(Child {
	                                     id: new_id(),
	                                     data: 0,
	                                 }));
	let root: Rc<RefCell<Box<View>>> =
		Rc::new(RefCell::new(Box::new(Root {
		                                  id: new_id(),
		                                  child: child.clone(),
		                              })));
	set_root("hover-overlay", root);
	start();
	(*child).borrow_mut().increment();
	(*child).borrow_mut().increment();
	(*child).borrow_mut().increment();
	(*child).borrow_mut().increment();
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
	data: u64,
}

impl View for Child {
	implement_id!();

	fn render_inner(&self, w: &mut String) {
		write!(w, "Hello world: {}", self.data).unwrap();
	}
}

impl Child {
	fn increment(&mut self) {
		self.data += 1;
		update(self);
	}
}
