use serde;
use serde::{Deserialize, Deserializer};
use serde::de::Visitor;
use std::convert::From;
use std::fmt;
use std::mem::transmute;

// Data of any post - either reply or OP
#[derive(Default, Deserialize, Clone)]
#[serde(default)]
pub struct Post {
	pub editing: bool,
	pub deleted: bool,
	pub banned: bool,
	pub sage: bool,
	pub sticky: bool,
	pub time: u64,
	pub id: u64,
	pub op: u64,
	#[serde(skip)]
	pub state: TextState,
	pub image: Option<Image>,
	pub body: String,
	pub name: Option<String>,
	pub trip: Option<String>,
	pub auth: Option<String>,
	pub flag: Option<String>,
	pub board: Option<String>,
	pub subject: Option<String>,
	pub links: Option<Vec<[u64; 2]>>,
	// TODO: Hash command enum
	// commands: Option<Vec<Command>>,
}

impl<'a> From<&'a Thread> for Post {
	// Copy post properties from a thread struct
	// TODO: Eliminate this, when switching to binary encoding
	fn from(t: &Thread) -> Post {
		macro_rules! copy {
			( $( $prop:ident ),* ) => (
				Post {
					$($prop: t.$prop.clone(),)*
				}
			)
		}
		copy!(
			editing,
			deleted,
			banned,
			sage,
			sticky,
			time,
			id,
			op,
			state,
			image,
			body,
			name,
			trip,
			auth,
			flag,
			board,
			subject,
			links
		)
	}
}

#[derive(Default, Deserialize, Clone)]
#[serde(rename_all = "camelCase")]
#[serde(default)]
pub struct Thread {
	pub editing: bool,
	pub deleted: bool,
	pub banned: bool,
	pub sage: bool,
	pub sticky: bool,
	pub time: u64,
	pub id: u64,
	pub op: u64,
	#[serde(skip)]
	pub state: TextState,
	pub image: Option<Image>,
	pub body: String,
	pub name: Option<String>,
	pub trip: Option<String>,
	pub auth: Option<String>,
	pub flag: Option<String>,
	pub board: Option<String>,
	pub subject: Option<String>,
	pub links: Option<Vec<[u64; 2]>>,
	// TODO: Hash command enum
	// commands: Option<Vec<Command>>,

	// Extra fields for OPs
	pub post_ctr: u64,
	pub image_ctr: u64,
	pub reply_time: u64,
	pub bump_time: u64,
	pub posts: Option<Vec<Post>>,
}

// Data of a file attached to a post
#[derive(Default, Deserialize, Clone)]
#[serde(default)]
#[serde(rename_all = "camelCase")]
pub struct Image {
	apng: bool,
	audio: bool,
	video: bool,
	spoiler: bool,
	#[serde(skip)]
	expanded: bool,
	#[serde(skip)]
	taller_than_viewport: bool,
	#[serde(skip)]
	revealed: bool,
	file_type: FileType,
	thumb_type: FileType,
	length: Option<u32>,
	size: u64,
	dims: [u16; 4],
	artist: Option<String>,
	title: Option<String>,
	#[serde(rename = "MD5")]
	md5: String,
	#[serde(rename = "SHA1")]
	sha1: String,
	name: String,
}

#[derive(Clone)]
#[repr(u8)]
pub enum FileType {
	JPG,
	PNG,
	GIF,
	WEBM,
	PDF,
	SVG,
	MP4,
	MP3,
	OGG,
	ZIP,
	SevenZIP,
	TARGZ,
	TARXZ,
	FLAC,
	NoFile,
	TXT,
}

impl Default for FileType {
	fn default() -> FileType {
		FileType::JPG
	}
}

impl<'a> Deserialize<'a> for FileType {
	fn deserialize<D>(des: D) -> Result<FileType, D::Error>
	where
		D: Deserializer<'a>,
	{
		des.deserialize_u8(FileTypeVisitor)
	}
}

// Custom deserialization for FileType enum
struct FileTypeVisitor;

impl<'a> Visitor<'a> for FileTypeVisitor {
	type Value = FileType;

	fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
		formatter.write_str("uint")
	}

	fn visit_u8<E>(self, value: u8) -> Result<FileType, E>
	where
		E: serde::de::Error,
	{
		Ok(unsafe { transmute(value) })
	}

	fn visit_i64<E>(self, value: i64) -> Result<FileType, E>
	where
		E: serde::de::Error,
	{
		Ok(unsafe { transmute(value as u8) })
	}

	fn visit_u64<E>(self, value: u64) -> Result<FileType, E>
	where
		E: serde::de::Error,
	{
		Ok(unsafe { transmute(value as u8) })
	}
}

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
#[derive(Default, Deserialize, Clone)]
pub struct TextState {
	spoiler: bool,
	quote: bool,
	last_line_empty: bool,
	code: bool,
	have_sync_watch: bool,
	i_dice: u32,
}
