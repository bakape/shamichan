use protocol::payloads::{FileType, Image};

#[derive(
	serde::Serialize,
	serde::Deserialize,
	Debug,
	Clone,
	Copy,
	Eq,
	PartialEq,
	std::hash::Hash,
	PartialOrd,
	Ord,
)]
#[serde(rename_all = "lowercase")]
pub enum Provider {
	Google,
	Yandex,
	IQDB,
	SauceNao,
	Trace,
	DesuArchive,
	ExHentai,
}

impl Provider {
	/// Return string identifier of this provider
	pub fn key(&self) -> &'static str {
		use Provider::*;

		match self {
			Google => "google",
			Yandex => "yandex",
			IQDB => "iqdb",
			SauceNao => "saucenao",
			Trace => "trace",
			DesuArchive => "desuarchive",
			ExHentai => "exhentai",
		}
	}

	/// Return short string symbol of this provider
	pub fn symbol(&self) -> &'static str {
		use Provider::*;

		match self {
			Google => "G",
			Yandex => "Y",
			IQDB => "IQDB",
			SauceNao => "SN",
			Trace => "T",
			DesuArchive => "DA",
			ExHentai => "EH",
		}
	}

	/// Return url for querying the image search provider, if available
	pub fn url(&self, img: &Image, img_url: &str) -> Option<String> {
		use FileType::*;
		use Provider::*;

		match self {
			DesuArchive => match img.common.file_type {
				JPEG
				| PNG
				| GIF
				| WEBM => format!(
					"https://desuarchive.org/_/search/image/{}",
					base64::encode(&img.md5)
				)
				.into(),
				_ => None,
			},
			ExHentai => match img.common.file_type {
				JPEG | PNG => format!(
					"http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash={}",
					hex::encode(&img.md5)
				)
				.into(),
				_ => None,
			},
			_ => format!(
				"{}{}",
				match self {
					Google =>
						"https://www.google.com/searchbyimage?image_url=",
					Yandex => "https://yandex.com/images/search?source=collections&rpt=imageview&url=",
					IQDB => "http://iqdb.org/?url=",
					SauceNao => "http://saucenao.com/search.php?db=999&url=",
					Trace => "https://trace.moe/?url=",
					_ => return None,
				},
				img_url,
			)
			.into(),
		}
	}
}
