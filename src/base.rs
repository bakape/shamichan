use std::fmt::Write;

pub trait ID<'a> {
	fn id(&self) -> &'a str;
}

// Base unit of manipulation
pub trait View<'a>: ID<'a> {
	fn render(&self) -> String;
}

// View with possible children
pub trait Parent<'a>: ID<'a> {
	type CH: View<'a>;

	fn render_outer(&self) -> String;
	fn children(&self) -> &mut Vec<Self::CH>;
}

impl<'a, P> View<'a> for P
    where P: Parent<'a>
{
	// Assumes to receive valid HTML or escaped text.
	fn render(&self) -> String {
		let s = self.render_outer();

		// Text node
		if s.chars().nth(0) != Some('<') {
			return s;
		}

		let mut w = String::with_capacity(1 << 10);
		let mut chars = s.chars().skip(1);
		w.push('>');

		// Extract the closing tag of the element
		let mut closing = String::with_capacity(16);
		closing += "</";
		while let Some(ch) = chars.next() {
			match ch {
				'0'...'9' | 'A'...'Z' | 'a'...'z' | '-' => {
					w.push(ch);
					closing.push(ch);
				}
				_ => {
					write!(w, " id=\"{}\"", self.id()).unwrap();
					w.push(ch);
					break;
				}
			}
		}

		// Drain the rest and remove closing tag
		w.extend(chars);
		closing.push('>');
		let len = w.len();
		w.truncate(len - closing.len());

		// Render all children
		for ch in self.children().iter() {
			w += &ch.render();
		}

		// Reapply closing tag
		w += &closing;

		w
	}
}
