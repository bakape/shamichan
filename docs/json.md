Public JSON API documentation. Refer to [common.md](common.md) for more
information.

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
| /json/extensions | GET | - | [fileTypes](common.md#filetypes) | Returns a map of the current filetype enums to their canonical extensions |
| /json/boardList | GET | - | [][BoardTitle](#boardtitle) | Returns an array of the currently created boards and their assigned titles |
| /uploadHash | POST | string{40} | string | Files can be inserted into a post without uploading the actual file, if it already exists on the server. To do this upload the hex-encoded SHA1 hash of the file you wish to insert into the post. If the file exists on the server a upload token is returned, otherwise response body is empty. Use this token in an [ImageRequest](common.md#imagerequest). |
| /upload | POST | form{"image": File} | string | Uploads a file in a form under the "image" field. Returns a token to be used in [ImageRequest](common.md#imagerequest) for allocating images to posts. |

## Thread

extends [Post](common.md#post)

| Field | Type | Required | Description |
|---|---|:---:|---|
| abbrev | bool | - | Specifies, if the thread is abbreviated and does not contain all of its replies |
| postCtr | uint | + | Number of posts in the thread |
| imageCtr | uint | + | Number of posts with images in the thread |
| replyTime | uint | + | Unix timestamp of the time of the last reply in the thread |
| bumpTime | uint | + | Unix timestamp of when the thread was last bumped to the top of the board |
| subject | string | + | Subject of the thread |
| board | string | + | Parent board of the thread |
| posts | [][Post](common.md#post) | + | Array of reply posts to the thread |

## StandalonePost
Additionally contains fields that define the posts parenthood

extends [Post](common.md#post)

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
