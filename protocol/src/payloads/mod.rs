pub mod post_body;

use hex_buffer_serde::{Hex, HexForm};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::Arc};

// Helper for big array serialization
big_array! { BigArray; }

/// Wrapper to enable logging and serialization
#[derive(Serialize, Deserialize, Clone)]
pub struct Signature(#[serde(with = "BigArray")] pub [u8; 512]);

impl std::fmt::Debug for Signature {
	fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
		write!(f, "{}", hex::encode(&self.0 as &[u8]))
	}
}

/// Authentication creds sent to the server during a handshake
#[derive(Serialize, Deserialize, Debug, Clone)]
pub enum Authorization {
	/// New public key registration
	NewPubKey(Vec<u8>),

	/// Key already persisted on the server
	Saved {
		/// ID of pub key on the server
		id: uuid::Uuid,

		/// Nonce to hash along with id
		nonce: [u8; 32],

		/// SHA3-256 signature of id + nonce
		signature: Signature,
	},
}

/// Authenticate with the server
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct HandshakeReq {
	/// Protocol version the client implements
	pub protocol_version: u16,

	/// Used to authenticate the client
	pub auth: Authorization,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct HandshakeRes {
	/// Key already saved in database. Need to confirm it's the same private key
	/// by sending a HandshakeReq with Authentication::Saved.
	pub need_resend: bool,

	/// ID of key on the server
	pub id: uuid::Uuid,
}

/// Request for creating a new thread
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ThreadCreationReq {
	pub subject: String,
	pub tags: Vec<String>,
	pub captcha_solution: Vec<u8>,
	pub opts: NewPostOpts,
}

/// Options for creating new posts (both OPs and replies)
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct NewPostOpts {
	pub name: String,
	pub trip: String,
	// TODO: staff titles
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ThreadCreationNotice {
	pub id: u64,
	pub subject: String,
	pub tags: Vec<String>,
	pub time: u32,
}

/// Request to insert a new post into a thread
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PostCreationReq {
	pub sage: bool,
	pub thread: u64,
	pub opts: NewPostOpts,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct PostCreationNotification {
	pub id: u64,
	pub thread: u64,
	pub time: u32,
	pub page: u32,
}

/// State of an open post. Used to diff the current state of the client
/// against the server feed's state.
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct OpenPost {
	pub has_image: bool,
	pub image_spoilered: bool,
	pub created_on: u32,
	pub thread: u64,
	pub body: Arc<post_body::Node>,
}

impl OpenPost {
	pub fn new(thread: u64, created_on: u32) -> Self {
		Self {
			created_on,
			thread,
			has_image: Default::default(),
			image_spoilered: Default::default(),
			body: Default::default(),
		}
	}
}

/// Feed initialization data
#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct FeedData {
	/// Thread this feed refers to
	pub thread: u64,

	/// Posts created in the last 16 minutes (open post CD + 1 minute to ensure
	/// there is no overlap due to latency).
	/// <post_id: post_creation_unix_timestamp>
	pub recent_posts: HashMap<u64, u32>,

	/// Posts currently open. Mapped by ID.
	pub open_posts: HashMap<u64, OpenPost>,
	// TODO: Applied moderation
}

/// Supported file types
#[derive(Serialize, Deserialize, Debug, Copy, Clone, PartialEq, Eq)]
pub enum FileType {
	JPEG,
	PNG,
	GIF,
	WEBM,
	PDF,
	SVG,
	MP4,
	MP3,
	OGG,
	ZIP,

	#[serde(rename = "7Z")]
	SevenZip,

	TGZ,
	TXZ,
	FLAC,

	#[serde(rename = "NO_FILE")]
	NoFile,

	TXT,
	WEBP,
	RAR,
	CBZ,
	CBR,
}

impl FileType {
	/// Return canonical extension for file type
	pub fn extension(&self) -> &'static str {
		use FileType::*;

		match self {
			JPEG => "jpg",
			PNG => "png",
			GIF => "gif",
			WEBP => "webp",
			MP3 => "mp3",
			MP4 => "mp4",
			WEBM => "webm",
			OGG => "ogg",
			PDF => "pdf",
			ZIP => "zip",
			SevenZip => "7z",
			TGZ => "tar.gz",
			TXZ => "tar.xz",
			FLAC => "flac",
			TXT => "txt",
			RAR => "rar",
			CBZ => "cbz",
			CBR => "cbr",
			SVG => "svg",
			NoFile => "",
		}
	}
}

/// Image data common to both binary and JSON representations
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ImageCommon {
	pub audio: bool,
	pub video: bool,

	pub file_type: FileType,
	pub thumb_type: FileType,

	pub width: u16,
	pub height: u16,
	pub thumb_width: u16,
	pub thumb_height: u16,

	pub duration: u32,
	pub size: u64,

	pub artist: Option<String>,
	pub title: Option<String>,

	pub name: String,
	pub spoilered: bool,
}

/// Image data JSON representation
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ImageJSON {
	#[serde(flatten)]
	pub common: ImageCommon,

	#[serde(with = "HexForm::<[u8; 20]>")]
	pub sha1: [u8; 20],

	#[serde(with = "HexForm::<[u8; 16]>")]
	pub md5: [u8; 16],
}

impl Into<Image> for ImageJSON {
	fn into(self) -> Image {
		Image {
			common: self.common,
			sha1: self.sha1,
			md5: self.md5,
		}
	}
}

/// Image data inserted into an open post
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Image {
	pub common: ImageCommon,
	pub sha1: [u8; 20],
	pub md5: [u8; 16],
}

/// Insert image into an open post
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct InsertImage {
	pub post: u64,
	pub image: Image,
}
