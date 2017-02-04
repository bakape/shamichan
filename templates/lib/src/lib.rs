mod matchers;

use matchers::{is_keyword, is_operator};
use std::{mem, slice};

// Needed to be able to compile to a bin package. WASM and asm.js targets do not
// currently support library packages.
#[allow(dead_code)]
fn main() {}

#[repr(C)]
pub struct Buffer {
	data: *const u8,
	size: usize,
}

impl Buffer {
	fn to_slice(&self) -> &[u8] {
		unsafe { slice::from_raw_parts(self.data, self.size) }
	}
	fn from_slice(data: &[u8]) -> Buffer {
		mem::forget(data);
		Buffer {
			data: data.as_ptr(),
			size: data.len(),
		}
	}
}

#[derive(PartialEq,PartialOrd)]
enum Type {
	Unmatched,
	Word,
	Quoted,
	DoubleQuoted,
	MultilineComment,
	LineComment,
}

// highlighting tags
const OPERATOR: &'static [u8] = b"<pre class=\"ms-operator\">";
const STRING: &'static [u8] = b"<pre class=\"ms-string\">";
const COMMENT: &'static [u8] = b"<pre class=\"ms-comment\">";

// convert valid UTF8 string to HTML with mostly language-agnostic minimalistic
// syntax highlighting
#[no_mangle]
pub extern "C" fn highlight_syntax(data: Buffer) -> Buffer {
	let text = data.to_slice();
	let mut buf = Vec::<u8>::with_capacity(data.size * 2);

	let mut typ = Type::Unmatched;
	let mut word = Vec::<u8>::with_capacity(64); // currently buffering word
	let mut prev = 0u8; // previous byte

	buf.extend_from_slice(b"<pre>");

	// close open tag
	macro_rules! close {
        () => (buf.extend_from_slice(b"</pre>"););
    }

	// check for keyword and flush any current word
	macro_rules! flush_word {
        () => {{
            let is = is_keyword(&word);
            if is {
                buf.extend_from_slice(OPERATOR);
            }
            buf.extend_from_slice(&word);
            if is {
                close!();
            }
            word.truncate(0);
        }};
    }

	for ch in text {
		let b = *ch;

		// terminate an escaping token, if matched
		macro_rules! terminate_if {
            ($check:expr) => {
                if $check {
                    close!();
                    typ = Type::Unmatched;
                }
            };
        }

		// start a new tag
		macro_rules! start {
            ($typ:expr, $head:expr) => {{
                if typ == Type::Word {
                    flush_word!();
                }
                typ = $typ;
                buf.extend_from_slice($head);
                buf.push(b);
            }};
        }

		// no matches and not a word start
		macro_rules! dump_unmatched {
            () => {{
                flush_word!();
                let is = is_operator(&b);
                if is {
                    buf.extend_from_slice(OPERATOR);
                }
                buf.push(b);
                if is {
                    close!();
                }
            }};
        }

		// possibly start a comment with a multiple character initializer
		macro_rules! multichar_comment {
            ($typ:expr) => {
                match prev {
                    b'/' => {
                        buf.pop();
                        typ = $typ;
                        buf.extend_from_slice(COMMENT);
                        buf.push(b'/');
                        buf.push(b);
                    },
                    _ => dump_unmatched!(),
                }
            }
        }

		if typ < Type::Word {
			buf.push(b);
		}
		match typ {
			// Continue escape tokens
			Type::Quoted => terminate_if!(b == b'\'' && prev != b'\\'),
			Type::DoubleQuoted => terminate_if!(b == b'"' && prev != b'\\'),
			Type::MultilineComment => terminate_if!(b == b'/' && prev == b'*'),
			Type::LineComment => terminate_if!(b == b'\n'),

			_ => {
				match b {
					65...90 | 97...122 => {
						word.push(b);
						typ = Type::Word;
					}
					b'\'' => start!(Type::Quoted, STRING),
					b'"' => start!(Type::DoubleQuoted, STRING),
					b'#' => start!(Type::LineComment, COMMENT),
					b'/' => multichar_comment!(Type::LineComment),
					b'*' => multichar_comment!(Type::MultilineComment),
					_ => dump_unmatched!(),
				}
			}
		};

		prev = b;
	}

	if typ == Type::Word {
		flush_word!();
	} else if typ != Type::Unmatched {
		close!();
	}
	close!();
	Buffer::from_slice(&buf)
}
