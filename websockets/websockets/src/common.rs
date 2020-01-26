use std::collections::{HashMap, HashSet};
use std::hash::Hash;

// Boxed error result type shorthand
pub type DynResult<T = ()> = Result<T, Box<dyn std::error::Error>>;

// Return a string as error
#[macro_export]
macro_rules! str_err {
	($msg:expr) => {
		return Err($msg.into());
	};
	($fmt:expr, $( $args:tt )* ) => {
		str_err!(format!($fmt, $($args)*))
    };
}

// Generate functions for safely accessing global variable behind a RWLock.
//
// $vis: accessor visibility
// $type: type to store; must implement Default
// $extra_init: extra initialization lambda to execute
#[macro_export]
macro_rules! gen_global_rwlock {
	($vis:vis, $type:ident, $extra_init:expr) => {
		static __ONCE: std::sync::Once = std::sync::Once::new();
		static mut __GLOBAL: Option<std::sync::RwLock<$type>> = None;

		fn __init() {
			__ONCE.call_once(|| {
				unsafe { __GLOBAL = Some(Default::default()) };
				$extra_init();
			});
		}

		// Open global for reading
		#[allow(unused)]
		$vis fn read<F, R>(cb: F) -> R
		where
			F: FnOnce(&$type) -> R,
		{
			__init();
			cb(&*unsafe { __GLOBAL.as_ref().unwrap().read().unwrap() })
		}

		// Open global for writing
		#[allow(unused)]
		$vis fn write<F, R>(cb: F) -> R
		where
			F: FnOnce(&mut $type) -> R,
		{
			__init();
			cb(&mut *unsafe { __GLOBAL.as_ref().unwrap().write().unwrap() })
		}
	};
	($type:ident) => {
		$crate::gen_global_rwlock!(,$type, || {});
	};
	($vis:vis, $type:ident) => {
		$crate::gen_global_rwlock!($vis, $type, || {});
	};
}

// Map of K to sets of V
#[derive(Clone)]
pub struct SetMap<K, V>(pub HashMap<K, HashSet<V>>)
where
	K: Hash + Eq + Clone,
	V: Hash + Eq + Clone;

impl<K, V> Default for SetMap<K, V>
where
	K: Hash + Eq + Clone,
	V: Hash + Eq + Clone,
{
	fn default() -> Self {
		Self(HashMap::new())
	}
}

impl<K, V> SetMap<K, V>
where
	K: Hash + Eq + Clone,
	V: Hash + Eq + Clone,
{
	pub fn new() -> Self {
		Default::default()
	}

	pub fn insert(&mut self, k: K, v: V) {
		match self.0.get_mut(&k) {
			Some(set) => {
				set.insert(v);
			}
			None => {
				let mut set = HashSet::new();
				set.insert(v);
				self.0.insert(k, set);
			}
		}
	}

	pub fn remove(&mut self, k: &K, v: &V)
	where
		K: Hash + Eq + Clone,
		V: Hash + Eq + Clone,
	{
		if let Some(set) = self.0.get_mut(k) {
			set.remove(v);
			if set.len() == 0 {
				self.0.remove(k);
			}
		}
	}

	pub fn remove_key(&mut self, k: &K)
	where
		K: Hash + Eq + Clone,
		V: Hash + Eq + Clone,
	{
		self.0.remove(k);
	}

	pub fn contains_key(&self, k: &K) -> bool {
		self.0.contains_key(k)
	}

	pub fn keys(&self) -> std::collections::hash_map::Keys<K, HashSet<V>> {
		self.0.keys()
	}

	pub fn get(&self, k: &K) -> Option<&HashSet<V>> {
		self.0.get(k)
	}
}
