use std::collections::{HashMap, HashSet};
use std::hash::Hash;

/// Maps of K to sets of V
#[derive(Debug, Clone)]
pub struct SetMap<K, V>(HashMap<K, HashSet<V>>)
where
	K: Hash + Eq,
	V: Hash + Eq;

impl<K, V> Default for SetMap<K, V>
where
	K: Hash + Eq,
	V: Hash + Eq,
{
	fn default() -> Self {
		Self(HashMap::new())
	}
}

impl<K, V> SetMap<K, V>
where
	K: Hash + Eq,
	V: Hash + Eq,
{
	pub fn insert(&mut self, k: K, v: V) {
		self.0.entry(k).or_default().insert(v);
	}

	pub fn remove(&mut self, k: &K, v: &V) -> bool {
		let mut removed = false;
		if let Some(set) = self.0.get_mut(k) {
			removed = set.remove(v);
			if set.is_empty() {
				self.0.remove(k);
			}
		}
		removed
	}

	#[inline]
	pub fn get(&self, k: &K) -> Option<&HashSet<V>> {
		self.0.get(k)
	}

	#[inline]
	pub fn get_mut(&mut self, k: &K) -> Option<&mut HashSet<V>> {
		self.0.get_mut(k)
	}

	#[inline]
	pub fn clear(&mut self) {
		self.0.clear()
	}

	#[inline]
	pub fn drain(
		&mut self,
	) -> std::collections::hash_map::Drain<'_, K, HashSet<V>> {
		self.0.drain()
	}

	#[inline]
	pub fn keys(&self) -> impl Iterator<Item = &K> {
		self.0.keys()
	}

	#[inline]
	pub fn values(&self) -> impl Iterator<Item = &V> {
		self.0.values().map(|s| s.iter()).flatten()
	}

	#[inline]
	pub fn is_empty(&self) -> bool {
		self.0.is_empty()
	}
}

impl<K, V> std::iter::FromIterator<(K, HashSet<V>)> for SetMap<K, V>
where
	K: Hash + Eq,
	V: Hash + Eq,
{
	#[inline]
	fn from_iter<T: IntoIterator<Item = (K, HashSet<V>)>>(iter: T) -> Self {
		Self(HashMap::<K, HashSet<V>>::from_iter(iter))
	}
}

/// Maps of K to sets of V and V to sets of K simultaneously
#[derive(Debug)]
pub struct DoubleSetMap<K, V>
where
	K: Hash + Eq + Clone,
	V: Hash + Eq + Clone,
{
	by_key: SetMap<K, V>,
	by_value: SetMap<V, K>,
}

impl<K, V> Default for DoubleSetMap<K, V>
where
	K: Hash + Eq + Clone,
	V: Hash + Eq + Clone,
{
	#[inline]
	fn default() -> Self {
		Self {
			by_key: Default::default(),
			by_value: Default::default(),
		}
	}
}

impl<K, V> DoubleSetMap<K, V>
where
	K: Hash + Eq + Clone,
	V: Hash + Eq + Clone,
{
	pub fn insert(&mut self, k: K, v: V) {
		self.by_key.insert(k.clone(), v.clone());
		self.by_value.insert(v, k);
	}

	#[inline]
	pub fn get_by_key(&self, k: &K) -> Option<&HashSet<V>> {
		self.by_key.get(k)
	}

	#[inline]
	pub fn get_by_value(&self, v: &V) -> Option<&HashSet<K>> {
		self.by_value.get(v)
	}

	pub fn remove_by_key(&mut self, k: &K) {
		if let Some(set) = self.by_key.0.remove(&k) {
			for v in set {
				self.by_value.remove(&v, &k);
			}
		}
	}

	pub fn remove_by_value(&mut self, v: &V) {
		if let Some(set) = self.by_value.0.remove(&v) {
			for k in set {
				self.by_key.remove(&k, &v);
			}
		}
	}

	pub fn remove_by_key_value(&mut self, k: &K, v: &V) {
		if let Some(set) = self.by_key.0.get_mut(&k) {
			if set.remove(&v) {
				self.by_value.remove(&v, &k);
			}
		}
	}
}

#[cfg(not(target_arch = "wasm32"))]
#[macro_export]
macro_rules! _debug_log_inner {
	($arg:expr) => {
		eprintln!("{}", &$arg);
	};
}

#[cfg(target_arch = "wasm32")]
#[macro_export]
#[rustfmt::skip]
macro_rules! _debug_log_inner {
	($arg:expr) => {{
		use wasm_bindgen::prelude::*;

		web_sys::console::log_1(&JsValue::from(&format!("{}", $arg)));
	}};
}

#[macro_export]
macro_rules! debug_log {
    ($arg:expr) => {
        if cfg!(debug_assertions) {
            $crate::_debug_log_inner!($arg);
        }
    };
	($label:expr, $arg:expr) => {
        debug_log!(format!("{}: {:?}", $label, &$arg));
    };
	($label:expr, $arg:expr, $($more:expr),+) => {
        debug_log!("{}: {:?}", $label, (&$arg $(, &$more)+));
	};
}
