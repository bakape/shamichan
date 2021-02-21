/// Simple min-queue with ability to remove arbitrary nodes by a unique key
pub struct Queue<T: PartialOrd> {
	head: Option<Box<Node<T>>>,
}

impl<T: PartialOrd> Default for Queue<T> {
	#[inline]
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
	/// Insert a new value into the queue
	pub fn insert(&mut self, val: T) {
		insert_before(&mut self.head, val);
	}

	/// Peek smallest value
	#[inline]
	pub fn peek<'n, 'l: 'n>(&'l self) -> Option<&'n T> {
		match &self.head {
			Some(n) => Some(&n.val),
			None => None,
		}
	}

	/// Remove smallest value
	pub fn pop(&mut self) -> Option<T> {
		self.head.take().map(|node| {
			self.head = node.next;
			node.val
		})
	}

	/// Remove a Node by unique key, if any
	pub fn remove(&mut self, key: &impl PartialEq<T>) {
		remove(&mut self.head, key);
	}

	/// Create iterator over the queue
	pub fn iter<'n, 'l: 'n>(&'l self) -> impl Iterator<Item = &'n T> {
		Iter::new(&self.head)
	}

	/// Create mutable iterator over the queue.
	///
	/// No changes that affect node order must be made.
	pub fn iter_mut<'n, 'l: 'n>(
		&'l mut self,
	) -> impl Iterator<Item = &'n mut T> {
		IterMut::new(&mut self.head)
	}
}

struct Node<T: PartialOrd> {
	val: T,
	next: Option<Box<Node<T>>>,
}

impl<T: PartialOrd> Node<T> {
	#[inline]
	fn new(val: T) -> Self {
		Self { val, next: None }
	}
}

/// Iterates over all nodes in the Queue
struct Iter<'a, T: PartialOrd> {
	next: Option<&'a Node<T>>,
}

impl<'a, T: PartialOrd> Iter<'a, T> {
	#[inline]
	fn new(first: &'a Option<Box<Node<T>>>) -> Self {
		Iter::<'a, T> {
			next: Self::unpack(first),
		}
	}

	#[inline]
	fn unpack(next: &'a Option<Box<Node<T>>>) -> Option<&'a Node<T>> {
		next.as_ref().map(|x| x.as_ref())
	}
}

impl<'a, T: PartialOrd> Iterator for Iter<'a, T> {
	type Item = &'a T;

	fn next(&mut self) -> Option<&'a T> {
		self.next.take().map(|n| {
			self.next = Self::unpack(&n.next);
			(&n.val).into()
		})
	}
}

/// Iterates over all nodes in the Queue mutably.
///
/// No changes that affect node order must be made.
struct IterMut<'a, T: PartialOrd> {
	next: Option<&'a mut Node<T>>,
}

impl<'a, T: PartialOrd> IterMut<'a, T> {
	#[inline]
	fn new(first: &'a mut Option<Box<Node<T>>>) -> Self {
		IterMut::<'a, T> {
			next: Self::unpack(first),
		}
	}

	#[inline]
	fn unpack(next: &'a mut Option<Box<Node<T>>>) -> Option<&'a mut Node<T>> {
		next.as_mut().map(|x| x.as_mut())
	}
}

impl<'a, T: PartialOrd> Iterator for IterMut<'a, T> {
	type Item = &'a mut T;

	fn next(&mut self) -> Option<&'a mut T> {
		self.next.take().map(|n| {
			self.next = Self::unpack(&mut n.next);
			(&mut n.val).into()
		})
	}
}
