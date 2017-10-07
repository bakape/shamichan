#![allow(dead_code)] // TEMP

mod options;

use brunhild::get_inner_html;
use libc::uint64_t;
use posts::{Post, Thread};
use serde_json;
use std::borrow::BorrowMut;
use std::cell::RefCell;
use std::collections::{BTreeMap, HashMap, HashSet};
use std::mem::transmute;
use std::os::raw::{c_char, c_int};
use std::slice;

thread_local!{
	static STATE: RefCell<State> = RefCell::new(State::default())
}

#[derive(Default)]
pub struct State {
    pub options: options::Options,
    pub thread: Option<ThreadState>,
    pub posts: HashMap<u64, Post>,
}

// Thread-specific state of the page
pub struct ThreadState {
    post_count: u64,
    image_count: u64,
    reply_time: u64,
    bump_time: u64,
}

pub fn load() -> Result<(), serde_json::Error> {
    with_state(|state| {
        state.options = options::load();

        // Parse post JSON into application state
        let s = get_inner_html("post-data");
        let mut threads = Vec::<u64>::with_capacity(15);
        if state.page.thread != 0 {
            threads.push(state.page.thread);
            let t: Thread = serde_json::from_str(&s)?;
            state.thread = Some(ThreadState {
                post_count: t.post_ctr,
                image_count: t.image_ctr,
                reply_time: t.reply_time,
                bump_time: t.bump_time,
            });
            extract_thread(state, &t);
        } else {
            let board: Vec<Thread> = serde_json::from_str(&s)?;
            state.thread = None;
            for t in board.iter() {
                threads.push(t.id);
                extract_thread(state, &t);
            }

            // TODO: Catalog pages

        }

        unsafe { load_db(threads.as_ptr(), threads.len() as c_int) };

        Ok(())
    })
}

// Extract thread post data from intermediary parsed JSON struct
fn extract_thread(state: &mut State, t: &Thread) {
    state.posts.insert(t.id, Post::from(t));
    if let Some(ref posts) = t.posts {
        for p in posts.iter() {
            let mut c = p.clone();
            c.op = t.id;
            state.posts.insert(p.id, c);
        }
    }
}

// Run function, with the state of the application as an argument
pub fn with_state<F, R>(func: F) -> R
where
    F: FnOnce(&mut State) -> R,
{
    STATE.with(|r| func(r.borrow_mut().borrow_mut()))
}
