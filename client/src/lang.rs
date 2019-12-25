use super::util;
use serde::Deserialize;
use std::collections::HashMap;
use std::fmt;
use std::fmt::Write;
use wasm_bindgen::JsCast;

#[derive(Deserialize, Default)]
struct LanguagePack {
	// One to one mapping of string literals
	pub literals: HashMap<String, String>,

	// Parametric format strings
	pub format_strings: HashMap<String, FormatStr>,

	// (label, title) tuples
	pub labels: HashMap<String, (String, String)>,
}

super::gen_global!(LanguagePack);

// Component of a localization formatting expression
enum Token {
	Text(String),
	Variable(String),
}

// Parsed format string used for localization
struct FormatStr(Vec<Token>);

struct TokenVisitor();

impl<'de> serde::de::Visitor<'de> for TokenVisitor {
	type Value = FormatStr;

	fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
		formatter.write_str("localization format string")
	}

	fn visit_str<E>(self, mut s: &str) -> Result<Self::Value, E>
	where
		E: serde::de::Error,
	{
		let mut out = Vec::new();
		while s.len() > 0 {
			if match s.chars().position(|b| b == '{') {
				None => true,
				Some(start) => {
					if start != 0 {
						out.push(Token::Text(s[..start].into()));
						s = &s[start..];
					}
					match s.chars().position(|b| b == '}') {
						None => true,
						Some(end) => {
							out.push(Token::Variable(s[1..end].into()));
							s = &s[end + 1..];
							false
						}
					}
				}
			} {
				out.push(Token::Text(s.into()));
				break;
			}
		}
		Ok(FormatStr(out))
	}
}

impl<'de> Deserialize<'de> for FormatStr {
	fn deserialize<D>(d: D) -> Result<FormatStr, D::Error>
	where
		D: serde::de::Deserializer<'de>,
	{
		d.deserialize_str(TokenVisitor())
	}
}

// Localize strings by key
#[macro_export]
macro_rules! localize {
	// Localize string literal
	($key:expr) => {
		$crate::lang::localize_literal($key)
	};

	// Insert key-value pairs into localization format string
	($key:expr, { $($k:expr => $v:expr),+ }) => {
		$crate::lang::localize_format($key, &[$(($k, $v),)+])
	};
}

// Localize string literal
pub fn localize_literal(key: &str) -> &'static str {
	with(|l| match l.literals.get(key) {
		Some(v) => v,
		None => "localization not found",
	})
}

// Insert key-value pairs into parsed localization format string
pub fn localize_format(key: &str, args: &[(&str, &str)]) -> String {
	with(|l| match l.format_strings.get(key) {
		Some(fmt) => {
			let mut w = String::new();
			for t in fmt.0.iter() {
				match t {
					Token::Text(t) => w += &t,
					Token::Variable(k) => {
						match args.iter().find(|kv| kv.0 == k) {
							Some(kv) => w += kv.1,
							None => {
								write!(w, "{{{}}}", k).unwrap();
							}
						}
					}
				};
			}
			w
		}
		None => format!("localization not found: {}", key),
	})
}

#[test]
fn test_localization() {
	with(|l| {
		l.format_strings.insert(
			"test".into(),
			serde_json::from_str(r#""that {name} a {adjective}""#).unwrap(),
		);
		l.literals.insert("test".into(), "anon a BWAAKA".into());
	});

	assert_eq!(
		localize!("test", {"name" => "anon", "adjective"=> "BWAAKA"}),
		String::from("that anon a BWAAKA")
	);
	assert_eq!(localize!("test"), "anon a BWAAKA");
}

fn query_selector_all_iter<F>(sel: &str, mut f: F) -> util::Result
where
	F: FnMut(&web_sys::Element) -> util::Result,
{
	let els = util::document().query_selector_all(sel)?;
	for i in 0..els.length() {
		f(&els.get(i).unwrap().dyn_into::<web_sys::Element>().unwrap())?;
	}

	Ok(())
}

pub async fn load_language_pack() -> util::Result {
	async fn run(l: &mut LanguagePack) -> util::Result {
		*l = serde_json::from_str(&String::from(
			wasm_bindgen_futures::JsFuture::from(
				js_sys::Reflect::get(&util::window(), &"language_pack".into())?
					.dyn_into::<js_sys::Promise>()?,
			)
			.await?
			.dyn_into::<js_sys::JsString>()?,
		))?;

		// Apply localization to static DOM elements
		query_selector_all_iter("[lang-content]", |el| {
			el.set_text_content(Some(localize! {
				&el
				.get_attribute("lang-content")
				.unwrap()
			}));
			Ok(())
		})?;
		query_selector_all_iter("[lang-title]", |el| {
			el.set_attribute(
				"title",
				localize!(&el.get_attribute("lang-title").unwrap()),
			)?;
			Ok(())
		})?;

		Ok(())
	}

	with(run).await
}
