# Public JSON API documentation

Note, that to minimize network payload fields at their null values are omitted.
For example a post containing `"editing":false` will have the editing field
omitted.

| URL | Type | Request payload | Response payload | Description |
|---|---|---|---|---|
| /json/all/ | GET | - | [][Thread](#thread) | Returns all threads from all boards complete with the last 5 replies as an array of [Thread](#thread) |
| /json/:board/ | GET | - | [][Thread](#thread) | Returns specific board specified by the `:board` parameter complete with the last 5 replies as an array of [Thread](#thread) |
| /json/all/catalog | GET | - | [][Thread](#thread) | Same as above, but does not return any replies |
| /json/:board/catalog | GET | - | [][Thread](#thread) | Same as above, but does not return any replies |
| /json/:board/:thread | GET | - | [Thread](#thread) | Returns a specific thread on a specific board. Accepts the `last=N` query parameter. `N` specifies the maximum amount of replies to return and can be either 5 or 100. |
| /json/post/:post | GET | - | [StandalonePost](#standalonepost) | Returns a specific post located in any thread or board by its numeric ID. |
| /json/config | GET | - | [Config](#config) | Returns the current public server configuration |
| /json/boardConfig/:board | GET | - | [BoardConfig](#boardconfig) | Returns public board-specific configurations for the specific board |
| /json/extensions | GET | - | [fileTypes](#filetypes) | Returns a map of the current filetype enums to their canonical extensions |
| /json/boardList | GET | - | [][BoardTitle](#boardtitle) | Returns an array of the currently created boards and their assigned titles |
| /uploadHash | POST | string{40} | string | Files can be inserted into a post without uploading the actual file, if it already exists on the server. To do this upload the hex-encoded SHA1 hash of the file you wish to insert into the post. If the file exists on the server a upload token is returned, otherwise response body is empty. Use this token in an [ImageRequest](#imagerequest). |
| /upload | POST | form{"image": File} | string | Uploads a file in a form under the "image" field. Returns a token to be used in [ImageRequest](#imagerequest) for allocating images to posts. |

## Post
Generic post object

| Field | Type | Required | Description |
|---|---|:---:|---|
| editing | bool | - | describes, if the post is still open and its text body editable by the original creator of the post |
| deleted | bool | - | specifies, the post has been deleted by a moderator |
| banned | bool | - | specifies, the poster was banned for this post by a moderator |
| sage | bool | - | specifies, if the poster explicitly disabled bumping the thread |
| time | uint | + | Unix timestamp of post creation |
| id | uint | + | ID number of post. Unique globally, including across boards. |
| body | string | + | text body of post |
| name | string | - | poster name |
| trip | string | - | poster tripcode |
| auth | string | - | signifies staff posting with staff title enabled; one of "admin", "owners", "moderators" or "janitors" |
| backlinks | [PostLinks](#postlinks) | - | posts linking to this post |
| links | [PostLinks](#postlinks) | - | posts this post is linking |
| commands | [[]Command](#command) | - | results of hash commands, such as #flip |
| image | [Image](#image) | - | uploaded file data |

## Image
Uploaded file data attached to post

| Field | Type | Required | Description |
|---|---|:---:|---|
| apng | bool | - | describes, if file is an animated PNG |
| audio | bool | - | describes, if the file contains audio |
| video | bool | - | Only used for mp4 and ogg uploads, which may or may not contain a video stream. Describes, if they do. |
| spoiler | bool | - | describes, if image thumbnail is spoilered |
| fileType | [fileTypes](#filetypes) | + | file type of the originally uploaded file |
| thumbType | [fileTypes](#filetypes) | + | file type of the generated thumbnail |
| length | uint | - | Length of stream in seconds. Only used for audio and video files. |
| size | uint | + | size of originally uploaded file in bytes |
| dims | [4]uint | + | 4 item array containing the dimensions of the uploaded file and its thumbnail - [width, height, thumbnail_width, thumbnail_height] |
| MD5 | string | + | MD5 hash of the originally uploaded file. Encoded to unpadded base64 URL encoding. |
| SHA1 | string | + | SHA1 hash of the originally uploaded file. Encoded to hex. |
| name | string | + | file name the user uploaded the file with without extension |

## fileTypes
Enum representing all available file types an uploaded file can be. These are
also the canonical file extensions of these types. The extensions of thumbnails
is `.png` for all file types, except for `.jpg`, in which case it is `.jpg`.

```
jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg, zip, "7z", "tar.gz", "tar.xz"
```

## PostLinks
Array of linked post and parent thread tuples - [][2]uint

## Command
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
| syncWatch | [5]uint | stores data of the synchronized time counter as [hours, minutes, seconds, start_time, end_time] |
| pyu | uint | increment generic global counter and store current value |
| pcount | uint | store current global counter without incrementing |

## Thread

extends [Post](#post)

| Field | Type | Required | Description |
|---|---|:---:|---|
| abbrev | bool | - | Specifies, if the thread is abbreviated and does not contain all of its replies |
| postCtr | uint | + | Number of posts in the thread |
| imageCtr | uint | + | Number of posts with images in the thread |
| replyTime | uint | + | Unix timestamp of the time of the last reply in the thread |
| bumpTime | uint | + | Unix timestamp of when the thread was last bumped to the top of the board |
| subject | string | + | Subject of the thread |
| board | string | + | Parent board of the thread |
| posts | [][Post](#post) | + | Array of reply posts to the thread |

## StandalonePost
Additionally contains fields that define the posts parenthood

extends [Post](#post)

| Field | Type | Required | Description |
|---|---|:---:|---|
| op | uint | + | ID of the parent thread |
| board | string | + | ID of the parent board |

## Config

| Field | Type | Description |
|---|---|---|
| captcha | bool | Specifies, if captchas are enabled |
| mature | bool | Specifies, if the site is intended for mature audiences only. Used to optionally display a warning. |
| defaultLang | string | Default HTML language setting in POSIX locale format  |
| defaultCSS | string | Name of default CSS theme |
| links | map[string]string | Map of external link references. For example a key-value pair of `"4chan":"https://4chan.org"` would mean links typed in as `>>>/4chan/` should point to that specific URL. |

## BoardConfig

| Field | Type | Description |
|---|---|---|
| readOnly | bool | Specifies, if thread and post creation has been disabled on this board |
| textOnly | bool | Specifies, if file upload has been disabled |
| forcedAnon | bool | Specifies, if poster names and tripcodes have been disabled |
| title | string | Title of the board |
| notice | string | Short notice from the board owner |
| rules | string | Rules of current board |

## BoardTitle

| Field | Type | Description |
|---|---|---|
| id | string | ID of the board |
| title | string | Title of the board |
