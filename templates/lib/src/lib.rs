// TODO: HTML escaping
// TODO: Handle UTF-8 code

mod matchers;

use matchers::{is_keyword, is_operator};
use std::{mem, slice};

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

struct Writer<'a> {
	typ: Type,
	start: usize,
	i: usize,
	text: &'a [u8],
	buf: Vec<u8>,
}

impl<'a> Writer<'a> {
	fn new(text: &'a [u8]) -> Self {
		Writer {
			typ: Type::Unmatched,
			start: 0,
			i: 0,
			text: text,
			buf: Vec::<u8>::with_capacity(text.len()),
		}
	}

	fn parse(&mut self) -> &[u8] {
		self.buf.extend_from_slice(b"<pre>");
		let mut prev = 0u8;

		for (i, ch) in self.text.iter().enumerate() {
			let b = *ch;
			self.i = i;

			if self.typ > Type::Word {
				self.buf.push(b);
			}
			match self.typ {
				// Continue escape tokens
				Type::Quoted => self.terminate_if(b == b'\'' && prev != b'\\'),
				Type::DoubleQuoted => {
					self.terminate_if(b == b'"' && prev != b'\\')
				}
				Type::MultilineComment => {
					self.terminate_if(b == b'/' && prev == b'*')
				}
				Type::LineComment => self.terminate_if(b == b'\n'),

				_ => {
					match b {
						65...90 | 97...122 => {
							self.typ = Type::Word;
						}
						b'\'' => self.open(b, Type::Quoted, STRING),
						b'"' => self.open(b, Type::DoubleQuoted, STRING),
						b'#' => self.open(b, Type::LineComment, COMMENT),
						b'/' => {
							self.multichar_comment(b, prev, Type::LineComment)
						}
						b'*' => {
							self.multichar_comment(b,
							                       prev,
							                       Type::MultilineComment)
						}
						_ => self.dump_unmatched(b),
					}
				}
			};

			prev = b;
		}

		if self.typ == Type::Word {
			self.flush_word();
		} else if self.typ != Type::Unmatched {
			self.close();
		}
		self.close();
		&self.buf
	}

	// start a new tag
	fn open(&mut self, b: u8, typ: Type, header: &[u8]) {
		if self.typ == Type::Word {
			self.flush_word();
		}
		self.typ = typ;
		self.buf.extend_from_slice(header);
		self.buf.push(b);
	}

	// close open tag
	fn close(&mut self) {
		self.buf.extend_from_slice(b"</pre>");
		self.start = self.i;
	}

	// terminate an escaping token, if matched
	fn terminate_if(&mut self, terminate: bool) {
		if terminate {
			self.close();
			self.typ = Type::Unmatched;
		}
	}

	// check for keyword and flush any current word
	fn flush_word(&mut self) {
		let word = &self.text[self.start..self.i];
		let is = is_keyword(&word);
		if is {
			self.buf.extend_from_slice(OPERATOR);
		}
		self.buf.extend_from_slice(&word);
		if is {
			self.close();
		}
	}

	// possibly start a comment with a multiple character initializer
	fn multichar_comment(&mut self, b: u8, prev: u8, typ: Type) {
		match prev {
			b'/' => {
				self.buf.pop();
				self.typ = typ;
				self.buf.extend_from_slice(COMMENT);
				self.buf.push(b'/');
				self.buf.push(b);
			}
			_ => self.dump_unmatched(b),
		}
	}

	// no matches and not a word start
	fn dump_unmatched(&mut self, b: u8) {
		self.flush_word();
		let is = is_operator(b);
		if is {
			self.buf.extend_from_slice(OPERATOR);
		}
		self.buf.push(b);
		self.start = self.i;
		if is {
			self.close();
		}
	}
}

// convert valid UTF8 string to HTML with mostly language-agnostic minimalistic
// syntax highlighting
#[no_mangle]
pub extern "C" fn highlight_syntax(data: Buffer) -> Buffer {
	Buffer::from_slice(Writer::new(data.to_slice()).parse())
}
