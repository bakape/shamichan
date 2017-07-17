pub use self::externs::get_inner_html;
use std::borrow::BorrowMut;
use std::cell::RefCell;

thread_local! {
	static MUTATIONS: RefCell<Vec<Mutation>> = RefCell::new(Vec::new());
}

// Single buffered mutation to be written to the dom
struct Mutation {
	id: String,
	data: MutationData,
}

// Data of a pending mutation
#[allow(non_camel_case_types)]
enum MutationData {
	// Insertions. Contain HTML strings.
	append(String),
	prepend(String),
	before(String),
	after(String),
	set_inner_html(String),
	set_outer_html(String),

	// Remove node
	remove,
}

macro_rules! define_mutators {
	( $( $id:ident ),* ) => (
		$(
			pub fn $id(parent_id: &str, html: &str) {
				push_mutation(parent_id, MutationData::$id(String::from(html)));
			}
		)*
	)
}

define_mutators!(
	set_outer_html,
	set_inner_html,
	append,
	prepend,
	before,
	after
);

// Remove a node by ID
pub fn remove(id: &str) {
	push_mutation(id, MutationData::remove);
}

// Push mutation to the stack to be executed on RAF
fn push_mutation(id: &str, data: MutationData) {
	// TODO: Intelligently deduplicate repeated set_inner_html and
	// set_outer_html. Will need to include checks for null elements in all
	// operations.
	with_mutations(|m| {
		m.push(Mutation {
			id: String::from(id),
			data: data,
		})
	});
}

fn with_mutations<F>(func: F)
where
	F: FnOnce(&mut Vec<Mutation>),
{
	MUTATIONS.with(|r| func(r.borrow_mut().borrow_mut()));
}

// Applies any buffered DOM mutations.
// This is registered to emscripten_set_main_loop by start().
// If you wish to use a different function for the main loop, call this in
// emscripten_set_main_loop with `fps = 0`.
pub extern "C" fn flush_mutations() {
	with_mutations(|mutations| {
		for mutation in mutations.iter() {
			let id = &mutation.id;
			match mutation.data {
				MutationData::append(ref html) => externs::append(id, &html),
				MutationData::prepend(ref html) => externs::prepend(id, &html),
				MutationData::before(ref html) => externs::before(id, &html),
				MutationData::after(ref html) => externs::after(id, &html),
				MutationData::set_inner_html(ref html) => {
					externs::set_inner_html(id, &html)
				}
				MutationData::set_outer_html(ref html) => {
					externs::set_outer_html(id, &html)
				}
				MutationData::remove => externs::remove(id),
			};
		}
		mutations.truncate(0);
	});
}

mod externs {
	// Define functions for writing to the DOM
	macro_rules! define_writers {
	( $( $id:ident ),* ) => (
		$(
			pub fn $id(id: &str, html: &str) {
				to_C_string!(id, {
					to_C_string!(html, {
						unsafe { ffi::$id(id, html) };
					})
				})
			}
		)*
	 )
}

	define_writers!(
		set_outer_html,
		set_inner_html,
		append,
		prepend,
		before,
		after
	);

	pub fn remove(id: &str) {
		to_C_string!(id, {
			unsafe { ffi::remove(id) };
		})
	}

	// Returns the inner HTML of an element by ID.
	// If no element found, an empty String is returned.
	// Usage of this function will cause extra repaints, so use sparingly.
	pub fn get_inner_html(id: &str) -> String {
		to_C_string!(id, {
			from_C_string!(ffi::get_inner_html(id))
		})
	}

	mod ffi {
		use libc::*;

		// Define external functions for writing to the DOM
		macro_rules! define_writers {
			( $( $id:ident ),* ) => (
				extern "C" {
					$( pub fn $id(id: *const c_char, html: *const c_char); )*
				}
			)
		}

		define_writers!(
			set_outer_html,
			set_inner_html,
			append,
			prepend,
			before,
			after
		);

		extern "C" {
			pub fn remove(id: *const c_char);
			pub fn get_inner_html(id: *const c_char) -> *mut c_char;
		}
	}
}
