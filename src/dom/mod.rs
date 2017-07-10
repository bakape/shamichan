pub use self::externs::get_inner_html;
use libc::*;
use std::borrow::BorrowMut;
use std::cell::RefCell;

mod externs;

static mut ID_COUNTER: u64 = 0;

thread_local! {
	static MUTATIONS: RefCell<Vec<Mutation>> = RefCell::new(Vec::new());
}

// Generate a new unique node ID
pub fn new_id() -> String {
	let s = format!("brunhild-{}", unsafe { ID_COUNTER });
	unsafe { ID_COUNTER += 1 };
	s
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
			#[allow(dead_code)]
			pub fn $id(parent_id: &str, html: &str) {
				push_mutation(parent_id, MutationData::$id(String::from(html)));
			}
		)*
	)
}

define_mutators!(set_outer_html,
                 set_inner_html,
                 append,
                 prepend,
                 before,
                 after);

// Remove a node by ID
#[allow(dead_code)]
fn remove(id: &str) {
	push_mutation(id, MutationData::remove);
}

// Push mutation to the stack to be executed on RAF
fn push_mutation(id: &str, data: MutationData) {
	with_mutations(|m| {
		               m.push(Mutation {
		                          id: String::from(id),
		                          data: data,
		                      })
		              });
}

fn with_mutations<F>(func: F)
	where F: FnOnce(&mut Vec<Mutation>)
{
	MUTATIONS.with(|r| func(r.borrow_mut().borrow_mut()));
}

// Applies any buffered DOM mutations.
// This is registered to emscripten_set_main_loop by start().
// If you wish to use a different function for the main loop, call this in
// emscripten_set_main_loop with `fps = 0`.
pub extern "C" fn flush_mutations() {
	// TODO: Intelligently deduplicate repeated set_inner_html and
	// set_outer_html. Will need to include checks for null elements in all
	// operations.

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

// Register flush_mutations() with emscripten event loop
pub fn start() {
	unsafe {
		emscripten_set_main_loop(flush_mutations, 0, 0);
	}
}

extern "C" {
	pub fn emscripten_set_main_loop(func: extern "C" fn(),
	                                fps: c_int,
	                                infinite: c_int);
}
