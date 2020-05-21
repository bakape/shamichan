// Simple min-queue with ability to remove arbitrary nodes by a unique key
pub struct Queue<T: PartialOrd> {
	head: Option<Box<Node<T>>>,
}

impl<T: PartialOrd> Default for Queue<T> {
	fn default() -> Self {
		Self { head: None }
	}
}

fn insert_before<T: PartialOrd>(current: &mut Option<Box<Node<T>>>, val: T) {
	match current {
		Some(current) => {
			if val < current.val {
				let mut new = Node::new(val);
				std::mem::swap(&mut new, current);
				current.next = Some(Box::new(new));
			} else {
				insert_before(&mut current.next, val);
			}
		}
		None => {
			*current = Some(Box::new(Node::new(val)));
		}
	}
}

fn remove<T: PartialOrd>(
	current: &mut Option<Box<Node<T>>>,
	key: &impl PartialEq<T>,
) {
	if let Some(mut node) = current.take() {
		if key.eq(&node.val) {
			*current = node.next;
		} else {
			remove(&mut node.next, key);
			*current = Some(node);
		}
	}
}

impl<T: PartialOrd> Queue<T> {
	// Insert a new value into the queue
	pub fn insert(&mut self, val: T) {
		insert_before(&mut self.head, val);
	}

	// Peek smallest value
	pub fn peek<'n, 'l: 'n>(&'l self) -> Option<&'n T> {
		match &self.head {
			Some(n) => Some(&n.val),
			None => None,
		}
	}

	// Remove smallest value
	pub fn pop(&mut self) -> Option<T> {
		self.head.take().map(|node| {
			self.head = node.next;
			node.val
		})
	}

	// Remove a Node by unique key, if any
	pub fn remove(&mut self, key: &impl PartialEq<T>) {
		remove(&mut self.head, key);
	}

	pub fn iter<'n, 'l: 'n>(&'l self) -> Iter<'n, T> {
		Iter {
			current: self.head.as_ref().map(|x| x.as_ref()),
		}
	}
}

struct Node<T: PartialOrd> {
	val: T,
	next: Option<Box<Node<T>>>,
}

impl<T: PartialOrd> Node<T> {
	fn new(val: T) -> Self {
		Self { val, next: None }
	}
}

// Iterates over all nodes in the Queue
pub struct Iter<'a, T: PartialOrd> {
	current: Option<&'a Node<T>>,
}

impl<'a, T: PartialOrd> Iterator for Iter<'a, T> {
	type Item = &'a T;

	fn next(&mut self) -> Option<&'a T> {
		match self.current {
			Some(n) => {
				self.current = n.into();
				(&n.val).into()
			}
			None => None,
		}
	}
}
