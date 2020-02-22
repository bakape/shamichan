use protocol::{FileType, Image};

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
	// Return string identifier of this provider
	pub fn key(&self) -> &'static str {
		match self {
			Self::Google => "google",
			Self::Yandex => "yandex",
			Self::IQDB => "iqdb",
			Self::SauceNao => "saucenao",
			Self::Trace => "trace",
			Self::DesuArchive => "desuarchive",
			Self::ExHentai => "exhentai",
		}
	}

	// Return short string symbol of this provider
	pub fn symbol(&self) -> &'static str {
		match self {
			Self::Google => "G",
			Self::Yandex => "Y",
			Self::IQDB => "IQDB",
			Self::SauceNao => "SN",
			Self::Trace => "T",
			Self::DesuArchive => "DA",
			Self::ExHentai => "EH",
		}
	}

	// Return url for querying the image search provider, if available
	pub fn url(&self, img: &Image, img_url: &str) -> Option<String> {
		match self {
			Self::DesuArchive => match img.common.file_type {
				FileType::JPEG
				| FileType::PNG
				| FileType::GIF
				| FileType::WEBM => format!(
					"https://desuarchive.org/_/search/image/{}",
					base64::encode(&img.md5)
				)
				.into(),
				_ => None,
			},
			Self::ExHentai => match img.common.file_type {
				FileType::JPEG | FileType::PNG => format!(
					"http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash={}",
					hex::encode(&img.md5)
				)
				.into(),
				_ => None,
			},
			_ => format!(
				"{}{}",
				match self {
					Self::Google =>
						"https://www.google.com/searchbyimage?image_url=",
					Self::Yandex => "https://yandex.com/images/search?source=collections&rpt=imageview&url=",
					Self::IQDB => "http://iqdb.org/?url=",
					Self::SauceNao => "http://saucenao.com/search.php?db=999&url=",
					Self::Trace => "https://trace.moe/?url=",
					_ => return None,
				},
				img_url,
			)
			.into(),
		}
	}
}
