use brunhild::set_inner_html;
use externs::local_storage;
use serde_json;
use std::fmt::Write;

const SELECTED_KEY: &'static str = "selectedBoards";
const CATALOG_KEY: &'static str = "pointToCatalog";

// Render interactive board navigation
pub fn init() -> serde_json::Result<()> {
	render()
}

// Get boards selected for displaying
fn get_selected() -> serde_json::Result<Vec<String>> {
	let s = local_storage::get(SELECTED_KEY);
	if s.is_empty() {
		return Ok(vec![]);
	}
	serde_json::from_str(&s)
}

// Return, if boards should link to the catalog, instead of the board page
fn get_point_to_catalog() -> serde_json::Result<bool> {
	let s = local_storage::get(CATALOG_KEY);
	if s.is_empty() {
		return Ok(false);
	}
	serde_json::from_str(&s)
}

fn render() -> serde_json::Result<()> {
	let selected = get_selected()?;
	let catalog = get_point_to_catalog()?;
	let mut w = String::with_capacity(1 << 10);
	w.push('[');

	write_link(&mut w, "all", catalog);
	for b in selected {
		w += " / ";
		write_link(&mut w, &b, catalog);
	}

	write!(w, "] [<a class=\"board-selection bold mono\">+</a>]").unwrap();

	set_inner_html("board-navigation", &w);
	Ok(())
}

// Write link to board to w
fn write_link(w: &mut String, board: &str, catalog: bool) {
	write!(w, "<a href=\"../{}/", board).unwrap();
	if catalog {
		*w += "catalog";
	}
	write!(w, "\">{}</a>", board).unwrap();
}
