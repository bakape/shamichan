/// Collection keeping a sorted list of the last 5 IDs
#[derive(Debug)]
pub struct Last5 {
	len: usize,

	/// The sixth slot is only used as insertion space for easy sorting
	arr: [u64; 6],
}

impl Last5 {
	/// Initialize the collection with 1 ID. The collection may never be empty.
	pub fn new(first: u64) -> Self {
		let mut s = Self {
			len: 1,
			arr: Default::default(),
		};
		s.arr[0] = first;
		s
	}

	/// Return the smallest ID in the collection
	pub fn min(&self) -> u64 {
		self.arr[0]
	}

	/// Return the largest ID in the collection
	pub fn max(&self) -> u64 {
		self.arr[self.len - 1]
	}

	// Push an ID to the collection, if it modifies the current list of last 5
	// IDs
	pub fn push(&mut self, id: u64) {
		if self.len < 5 {
			self.len += 1;
		} else if id > self.max() {
			return;
		}
		self.arr[self.len] = id;
		self.arr[..self.len + 1].sort_unstable();
	}
}
