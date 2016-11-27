Common types used both in the WebSocket and JSON APIs

Note that to minimize network payload fields at their null values are omitted.
For example a post containing `"editing":false` will have the editing field
omitted.

##Post
Generic post object

| Field | Type | Required | Description |
|---|---|:---:|---|
| editing | bool | - | describes, if the post is still open and its text body editable by the original creator of the post |
| time | uint | + | Unix timestamp of post creation |
| id | uint | + | ID number of post. Unique globally, including across boards. |
| body | string | + | text body of post |
| name | string | - | poster name |
| trip | string | - | poster tripcode |
| email | string | - | poster email |
| backlinks | [PostLinks](#postlinks) | - | posts linking to this post |
| links | [PostLinks](#postlinks) | - | posts this post is linking |
| commands | [[]Command](#command) | - | results of hash commands, such as #flip |
| image | [Image](#image) | - | uploaded file data |

##Image
Uploaded file data attached to post

| Field | Type | Required | Description |
|---|---|:---:|---|
| apng | bool | - | describes, if file is an animated PNG |
| audio | bool | - | describes, if the file contains audio |
| video | bool | - | Only used for mp4 and ogg uploads, which may or may not contain a video stream. Describes, if they do. |
| spoiler | bool | - | describes, if image thumbnail is spoilered |
| fileType | [fileTypes](#filetypes) | + | file type of the originally uploaded file |
| length | uint | - | Length of stream in seconds. Only used for audio and video files. |
| size | uint | + | size of originally uploaded file in bytes |
| dims | [4]uint | + | 4 item array containing the dimensions of the uploaded file and its thumbnail - [width, height, thumbnail_width, thumbnail_height] |
| MD5 | string | + | MD5 hash of the originally uploaded file |
| SHA1 | string | + | SHA1 hash of the originally uploaded file |
| name | string | + | file name the user uploaded the file with without extension |

##fileTypes
Enum representing all available file types an uploaded file can be. These are
also the canonical file extensions of these types. The extensions of thumbnails
is `.png` for all file types, except for `jpg`, in which case it is `.jpg`.

```
jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg, zip, "7z", "tar.gz", "tar.xz"
```

##PostLinks
Map of linked post IDs to their parenthood data. Each map key contains the
following object:

| Field | Type | Required | Description |
|---|---|:---:|---|
| board | string | + | Parent board of the linked post |
| op | uint | + | Parent thread of the linked post |

##Command
Results of an executed hash command. Several different object types implement
this common interface and cary data appropriate to their command type. The
"type" field defines which type of command is stored, according to enum:

```
dice, flip, eightBall, syncWatch, pyu, pcount
```
The "val" field contains the following data for each command type:

| enum | Value type | Description |
|---|---|---|
| dice | []uint | Array of dice rolls. Maximum number of rolls is 10 and each roll can not exceed 100 |
| flip | bool | coin flip |
| eightBall | string | stores one of several predefined string messages randomly |
| syncWatch | ? | syncwatch not yet implemented and spec not finalized |
| pyu | uint | increment generic global counter and store current value |
| pcount | uint | store current global counter without incrementing |
