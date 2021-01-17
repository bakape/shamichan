pub mod post_body;

use hex_buffer_serde::{Hex, HexForm};
use post_body::Node;
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
#[serde(rename_all = "snake_case")]
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
pub enum PubKeyStatus {
	/// Key accepted. Handshake complete.
	Accepted,

	/// Key already saved in database. Need to confirm it's the same private key
	/// by sending a HandshakeReq with Authentication::Saved.
	NeedResend,

	/// Key not found in database. Need to send Authentication::NewPubKey to
	/// register it.
	NotFound,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct HandshakeRes {
	/// ID of key on the server
	pub id: uuid::Uuid,

	/// Public key status on the server
	pub status: PubKeyStatus,
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
	// TODO: staff titles
}

/// Additional options common to both OP and reply creation
#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct PostCreationOpts {
	pub name: Option<String>,
	pub trip: Option<String>,
	pub flag: Option<String>,
}

/// Additional options for reply creation
#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct ReplyCreationOpts {
	pub sage: bool,
	#[serde(flatten)]
	pub post_opts: PostCreationOpts,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct ThreadCreationNotice {
	pub id: u64,
	pub subject: String,
	pub tags: Vec<String>,
	pub time: u32,
	pub opts: PostCreationOpts,
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

/// Post from a thread
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Post {
	pub id: u64,
	pub page: u32,
	pub thread: u64,

	pub created_on: u32,
	pub open: bool,

	#[serde(flatten)]
	pub opts: ReplyCreationOpts,

	/// Post text body. Wrapped in an Arc to enable cheap copying on both the
	/// server and client
	pub body: Arc<Node>,

	pub image: Option<Image>,
}

impl Post {
	/// Create a new empty Post
	pub fn new(
		id: u64,
		thread: u64,
		page: u32,
		created_on: u32,
		opts: ReplyCreationOpts,
	) -> Self {
		Self {
			id,
			thread,
			page,
			created_on,
			open: true,
			opts,
			body: Default::default(),
			image: None,
		}
	}

	/// Create a new empty OP
	pub fn new_op(id: u64, created_on: u32, opts: PostCreationOpts) -> Self {
		Self::new(
			id,
			id,
			0,
			created_on,
			ReplyCreationOpts {
				sage: false,
				post_opts: opts,
			},
		)
	}
}

/// Thread information container
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Thread {
	/// Unique thread ID
	pub id: u64,

	/// Unix timestamp of thread creation time
	pub created_on: u32,

	/// Unix timestamp of the last time the thread was bumped
	pub bumped_on: u32,

	/// Thread subject
	pub subject: String,

	/// Tags applied to thread
	pub tags: Vec<String>,

	/// Number of page sin the thread
	pub page_count: u32,

	/// Number of posts in the thread, including the OP
	pub post_count: u64,

	/// Number of images in the thread
	pub image_count: u64,
}

impl Thread {
	/// Create a new thread with 1 empty OP
	pub fn new(
		id: u64,
		created_on: u32,
		subject: String,
		tags: Vec<String>,
	) -> Self {
		Self {
			id,
			subject,
			tags,
			created_on,
			page_count: 1,
			bumped_on: created_on,
			post_count: 1,
			image_count: 0,
		}
	}
}

/// A thread and it's posts flattened into a single structure
#[derive(Serialize, Deserialize, Debug)]
pub struct ThreadWithPosts {
	#[serde(flatten)]
	pub thread_data: Thread,

	pub posts: HashMap<u64, Post>,
}

/// Posts of a single immutable thread page
#[derive(Serialize, Deserialize, Debug)]
pub struct ImmutablePage {
	pub thread: u64,
	pub page: u32,
	pub posts: Vec<Post>,
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

/// Image data inserted into a open post
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Image {
	#[serde(with = "HexForm::<[u8; 20]>")]
	pub sha1: [u8; 20],
	#[serde(with = "HexForm::<[u8; 16]>")]
	pub md5: [u8; 16],

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

/// Request to insert image into an open post
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct InsertImage {
	pub post: u64,
	pub image: Image,
}
