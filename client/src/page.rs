use super::state::State;
use super::util;
use brunhild::{ElementOptions, Node};

// Fully render or rerender core central page content
pub fn render(s: &mut State) -> util::Result {
	s.views.aside_top.patch(Node::with_children(
		&ElementOptions::with_attrs(
			"section",
			&[&("class", "aside-container")],
		),
		vec![],
	))?;
	s.views.aside_bottom.patch(Node::with_children(
		&ElementOptions::with_attrs(
			"section",
			&[&("class", "aside-container")],
		),
		vec![],
	))?;

	Ok(())
}
