use super::state::State;
use super::util;

// Fully render or rerender core central page content
pub fn render(s: &mut State) -> util::Result {
	s.views.aside_top.patch(element!(
		"section",
		{"class" => "aside-container"},
		if s.thread == 0 {
			vec![]
		} else {
			vec![]
		}
	))?;
	s.views.aside_bottom.patch(element!(
		"section",
		{"class" => "aside-container"},
		if s.thread == 0 {
			vec![]
		} else {
			vec![]
		}
	))?;

	Ok(())
}
