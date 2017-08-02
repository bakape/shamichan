#![allow(dead_code)] // TEMP

use serde;
use serde::{Deserialize, Deserializer};
use serde::de::Visitor;
use std::convert::From;
use std::fmt;
use std::mem::transmute;

// Data of any post - either reply or OP
#[derive(Default, Deserialize, Clone)]
#[allow(non_snake_case)]
pub struct Post {
	#[serde(default)]
	pub editing: bool,
	#[serde(default)]
	pub deleted: bool,
	#[serde(default)]
	pub banned: bool,
	#[serde(default)]
	pub sage: bool,
	#[serde(default)]
	pub sticky: bool,
	pub time: u64,
	pub id: u64,
	#[serde(default)]
	pub op: u64,
	#[serde(default)]
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

#[allow(non_snake_case)]
#[derive(Deserialize, Clone)]
pub struct Thread {
	#[serde(default)]
	pub editing: bool,
	#[serde(default)]
	pub deleted: bool,
	#[serde(default)]
	pub banned: bool,
	#[serde(default)]
	pub sage: bool,
	#[serde(default)]
	pub sticky: bool,
	pub time: u64,
	pub id: u64,
	#[serde(default)]
	pub op: u64,
	#[serde(default)]
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
	pub postCtr: u64,
	pub imageCtr: u64,
	pub replyTime: u64,
	pub bumpTime: u64,
	pub posts: Option<Vec<Post>>,
}

// Data of a file attached to a post
#[derive(Default, Deserialize, Clone)]
#[allow(non_snake_case)]
pub struct Image {
	#[serde(default)]
	apng: bool,
	#[serde(default)]
	audio: bool,
	#[serde(default)]
	video: bool,
	#[serde(default)]
	spoiler: bool,
	#[serde(default)]
	expanded: bool,
	#[serde(default)]
	taller_than_viewport: bool,
	#[serde(default)]
	revealed: bool,
	fileType: FileType,
	thumbType: FileType,
	length: Option<u32>,
	size: u64,
	dims: [u16; 4],
	artist: Option<String>,
	title: Option<String>,
	MD5: String,
	SHA1: String,
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
