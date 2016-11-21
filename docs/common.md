Common types used both in the WebSocket and JSON APIs

Described in
[TypeScript-like type notation](https://www.typescriptlang.org/docs/handbook/interfaces.html)
with the following extensions:
- `int` signed 64 bit integer
- `uint` unsigned 64 bit integer
- `string{30}` string with a maximum allowed length of 30

Note that to minimize network payload fields at their null values are omitted.
For example a post containing `"editing":false` will have the editing field
omitted.

##Post
Generic post object
```
type Post = {
	// Defines, if the post is still open and its text body editable by the
	// original creator of the post
	editing?: boolean

	// Uploaded file data
	image?: Image

	// Unix timestamp of post creation
	time: uint

	// ID number of post. Unique globally, including across boards.
	id: uint

	// Text body of post
	body: string

	// Poster name
	name?: string

	// Poster tripcode
	trip?: string

	// Poster email address
	email?: string

	// Posts linking to this post
	backlinks?: PostLinks

	// Posts this post is linking
	links?: PostsLinks

	// Results of hash commands, such as #flip
	commands?: Command[]
}
```

##fileTypes
Enum representing all available file types an uploaded file can be. These are
also the canonical file extensions of these types. The extensions of thumbnails
is `.png` for all file types, except for `jpg`, in which case it is `.jpg`.
```
enum fileTypes {
	jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg, zip, "7z", "tar.gz", "tar.xz",
}
```

##Image
Uploaded file data attached to post
```
type Image = {
	// Defines, if file is an animated PNG
	apng: boolean

	// Defines, if the file contains audio
	audio: boolean

	// Only used for mp4 and ogg uploads, which may or may not contain a video
	// stream. Defines, if they do.
	video: boolean

	// Image is spoilered
	spoiler: boolean

	// File type of the originally uploaded file
	fileType: fileTypes

	// Length of stream in seconds. Only used for audio and video files.
	length?: uint

	// Size of originally uploaded file in bytes
	size: uint

	// 4-tuple containing the dimensions of the uploaded file
	// [width, height, thumbnail_width, thumbnail_height]
	dims: [uint, uint, uint, uint]

	// MD5 hash of the originally uploaded file
	MD5: string

	// SHA1 hash of the originally uploaded file
	SHA1: string

	// File name the user uploaded the file with without extension
	name: string
}
```

##PostLinks
Map of linked post IDs to their parenthood data
```
type PostLinks = { [id: uint]: PostLink)}

type PostLink = {
	// Parent board of the linked post
	board: string

	// Parent thread of the linked board
	op: uint
}
```

##Command
Results of an executed hash command. Several different object types implement
this common interface and cary data appropriate to their command type.
```
// Types of hash command entries
enum commandType { dice, flip, eightBall, syncWatch, pyu, pcount }

// Common interface
interface Command {
	// Carries the type of the object
	type: commandType
	val: uint|uint[]|bool|string
}

// Dice rolls
type Dice = {
	type = commandType.dice

	// Array of dice rolls. Maximum number of rolls is 10 and each roll can not
	// exceed 100
	vals: uint[]
}

// Coin flip
type Flip = {
	type = commandType.flip

	// Result of coin flip
	val: boolean
}

// Eightball prints one of several available string messages
type EightBall = {
	type = commandType.eightBall

	// Randomly chosen message
	val: string
}

// Syncwatch not yet implemented and spec not finalized

// Increment generic global counter
type Pyu = {
	type = commandType.pyu

	// Current value of counter after incrementing
	val: uint
}

// Print current global counter without incrementing
type Pcount = {
	type = commandType.pcount

	// Value of counter
	val: uint
}
```
