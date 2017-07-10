
// Data of any post - either reply or OP
#[derive(Default, Deserialize)]
#[allow(non_snake_case)]
pub struct Post {
	editing: bool,
	deleted: bool,
	banned: bool,
	sage: bool,
	sticky: bool,
	time: u64,
	id: u64,
	op: u64,
	state: TextState,
	image: Option<Image>,
	body: String,
	name: Option<String>,
	trip: Option<String>,
	auth: Option<String>,
	flag: Option<String>,
	board: Option<String>,
	subject: Option<String>,
	links: Option<Vec<[u64; 2]>>,
	// TODO: Hash command enum
	// commands: Option<Vec<Command>>,
}

#[allow(non_snake_case)]
#[derive(Deserialize)]
pub struct Thread {
	editing: bool,
	deleted: bool,
	banned: bool,
	sage: bool,
	sticky: bool,
	time: u64,
	id: u64,
	op: u64,
	state: TextState,
	image: Option<Image>,
	body: String,
	name: Option<String>,
	trip: Option<String>,
	auth: Option<String>,
	flag: Option<String>,
	board: Option<String>,
	subject: Option<String>,
	links: Option<Vec<[u64; 2]>>,

	// Extra fields for OPs
	postCtr: u64,
	imageCtr: u64,
	replyTime: u64,
	bumpTime: u64,
	posts: Option<Vec<Post>>,
}

#[derive(Deserialize)]
pub struct Board(Vec<Thread>);

// Data of a file attached to a post
#[derive(Default, Deserialize)]
#[allow(non_snake_case)]
pub struct Image {
	apng: bool,
	audio: bool,
	video: bool,
	spoiler: bool,
	expanded: bool,
	tallerThanViewport: bool,
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

#[derive(Deserialize)]
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

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
#[derive(Default, Deserialize)]
pub struct TextState {
	spoiler: bool,
	quote: bool,
	last_line_empty: bool,
	code: bool,
	have_sync_watch: bool,
	i_dice: u32,
}
