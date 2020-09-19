pub mod post_body;

use hex_buffer_serde::{Hex, HexForm};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::Arc};

/// Define a public payload struct with public fields
macro_rules! payload {
    (
		$(#[$struct_meta:meta])*
		$name:ident {
			$(
				$(#[$field_meta:meta])*
				$field:ident: $t:ty,
			)*
		}
	) => {
		$(#[$struct_meta])*
        #[derive(Serialize, Deserialize, Debug, Clone)]
        pub struct $name {
            $(
				$(#[$field_meta])*
				pub $field: $t
			),*
        }
	}
}

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

payload! {
	/// Authenticate with the server
	HandshakeReq {
		/// Protocol version the client implements
		protocol_version: u16,

		/// Used to authenticate the client
		auth: Authorization,
	}
}

payload! {
	HandshakeRes {
		/// Key already saved in database. Need to confirm it's the same private key
		/// by sending a HandshakeReq with Authentication::Saved.
		need_resend: bool,

		/// ID of key on the server
		id: uuid::Uuid,
	}
}

payload! {
	/// Request for creating a new thread
	ThreadCreationReq {
		subject: String,
		tags: Vec<String>,
		captcha_solution: Vec<u8>,
		opts: NewPostOpts,
	}
}

payload! {
	/// Options for creating new posts (both OPs and replies)
	NewPostOpts {
		name: String,
		// TODO: staff titles
	}
}

payload! {
	ThreadCreationNotice {
		id: u64,
		subject: String,
		tags: Vec<String>,
		time: u32,
	}
}

payload! {
	/// Request to insert a new post into a thread
	PostCreationReq {
		sage: bool,
		thread: u64,
		opts: NewPostOpts,
	}
}

payload! {
	PostCreationNotification {
		id: u64,
		thread: u64,
		time: u32,
		page: u32,
	}
}

payload! {
	/// State of an open post. Used to diff the current state of the client
	/// against the server feed's state.
	OpenPost {
		has_image: bool,
		image_spoilered: bool,
		created_on: u32,
		thread: u64,
		body: Arc<post_body::Node>,
	}
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

payload! {
	/// Feed initialization data
	#[derive(Default)]
	FeedData {
		/// Thread this feed refers to
		thread: u64,

		/// Posts created in the last 16 minutes (open post CD + 1 minute to ensure
		/// there is no overlap due to latency).
		/// <post_id: post_creation_unix_timestamp>
		recent_posts: HashMap<u64, u32>,

		/// Posts currently open. Mapped by ID.
		open_posts: HashMap<u64, OpenPost>,

		// TODO: Applied moderation
	}
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

payload! {
	/// Image data common to both binary and JSON representations
	ImageCommon {
		audio: bool,
		video: bool,

		file_type: FileType,
		thumb_type: FileType,

		width: u16,
		height: u16,
		thumb_width: u16,
		thumb_height: u16,

		duration: u32,
		size: u64,

		artist: Option<String>,
		title: Option<String>,

		name: String,
		spoilered: bool,
	}
}

payload! {
	/// Image data JSON representation
	ImageJSON {
		#[serde(flatten)]
		common: ImageCommon,

		#[serde(with = "HexForm::<[u8; 20]>")]
		sha1: [u8; 20],

		#[serde(with = "HexForm::<[u8; 16]>")]
		md5: [u8; 16],
	}
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

payload! {
	/// Image data inserted into an open post
	Image {
		common: ImageCommon,
		sha1: [u8; 20],
		md5: [u8; 16],
	}
}

payload! {
	/// Insert image into an open post
	InsertImage {
		post: u64,
		image: Image,
	}
}
