Documentation of the WebSocket API. For commonly used JSON types in the API see
[common.md](common.md).

- All communications are done through the `/socket` relative address
- The API only uses textual WebSocket frames for communication
- Only one message is transmitted per frame
- Each frame starts with two bytes with the ASCII-encoded message number. If the
message number is single digit, it must be padded with a leading zero.
- The first message after establishing a WebSocket connection to the server
must always be "synchronize".
- All complex payloads, such as JSON objects are JSON stringified

# Server to client

| Code | Name | Payload type | Description |
|:---:|---|---|---|
| 0 | invalid | string | Convey an unrecoverable error. Only used on client protocol violations or server errors. The connection is terminated after writing this message. You should either fix your client implementation or report a server bug, if this message is encountered. |
| 2 | insertPost | [Post](common.md#post) | Post insertion into the thread |
| 3 | append | [2]uint | Append a character to the current line of the post. The first array item is the ID of the target post. The seconds is a character encoded as UTF-8 character code. |
| 4 | backspace | uint | Remove one character from the end of the line of the post specified by ID. |
| 5 | splice | [SpliceMessage](#splicemessage) | Splice the current open line. Used for all text mutations, that are neither "append" or "backspace". |
| 6 | closePost | uint | Close the post specified by ID. This message may be received for already closed posts, due to asynchronous nature of the eventual synchronization algorithm. |
| 7 | backlink | [3]uint | Add a backlink to the post. A tuple of [target_post, linked_post, linked_post_parent_thread] |
| 8 | insertImage | [ImageMessage](#imagemessage) | Insert an image into an open post. |
| 9 | spoiler | uint | Spoiler the image of the post specified by ID |
| 10 | deletePost | uint | Delete a post specified by ID |
| 11 | banned | uint | Notifies the specific post was banned |
| 30 | synchronize | uint | Response to a synchronization request. Returns the current synchronization counter of the update feed |
| 31 | reclaim | uint | Response to a request to reclaim a post lost after disconnecting from the server. 0 denotes success and the client is henceforth able to write to said post, as before the disconnect.1 denotes the post is unrecoverable. |
| 32 | postID | int | Returns the post ID of the client's freshly allocated post. A response to a post or thread insertion request. -1 denotes invalid captcha. |
| 33 | concat | * | Contains several null-byte concatenated messages. Used for limiting the rate of update frames sent from the server. |
| 34 | NOOP | - | Invokes no operation on the server. Can be used as a connectivity test. |
| 35 | syncCount | uint | Sends the current unique connected IP count |
| 36 | serverTime | uint | Sends the server's current time's Unix timestamp. Can be used for approximating the clock skew between the server and client. |
| 37 | redirect | string | Notifies the client it has been redirected to the specific board page. Any open posts have been closed and after receiving this message the client is already considered to be synchronized to the board. If the client does comply an error will be raised the client will be disconnected. |

## SpliceMessage

extends [SpliceRequest](#splicerequest)

| Field | Type | Required | Description |
|---|---|:---:|---|
| id | uint | + | ID of the target post |

## ImageMessage
extends [Image](common.md#image)

| Field | Type | Required | Description |
|---|---|:---:|---|
| id | uint | + | ID of the target post |

# Client to server

Some fields of the string type have a maximum allowed length to prevent abuse.
It is defined in curly brackets after the string type. For example `string{30}`
denotes a string literal of maximum 30 bytes allowed length.

| Code | Name | Payload Type | Description |
|:---:|---|---|---|
| 1 | insertThread | [ThreaCreationRequest](#threadcreationrequest) | Request to create a new thread. After submitting the post is considered open and may be written to. |
| 2 | insertPost | [ReplyCreationRequest](#replycreationrequest) | Request to create a new reply. After submitting the post is considered open and may be written to. |
| 3 | append | uint | Append a character to the current line of the post. Encoded as UTF-8 character code. |
| 4 | backspace | - | Remove one character from the end of the current line. Does not contain any payload. |
| 5 | splice | [SpliceRequest](#splicerequest) | Splice the current open line. Used for all text mutations, that are neither "append" or "backspace". |
| 6 | closePost | - | Close the current open post. Does not contain any payload. |
| 8 | insertImage | [ImageRequest](#imagerequest) | Allocate an image to an already open post. |
| 30 | synchronize | [SyncRequest](#syncrequest) | Synchronize to a specific thread or board update feed. |
| 31 | reclaim | [ReclaimRequest](#reclaimrequest) | Reclaim an open post after losing connection to the server. Note that only open posts can be reclaimed and open posts are automatically closed 30 minutes after opening. |
| 43 | NOOP | - | No operation message. No payload. Can be used as a pseudo ping, if your WebSocket API does not expose pings. |

## ImageRequest
Request to allocate a file to a post. Note that allocation requests on boards
set to text-only will be ignored.

| Field | Type | Required | Description |
|---|---|:---:|---|
| spoiler | bool | - | Defines, if the image thumbnail should be spoilered  |
| token | string{127} | + | Allocation token retrieved from either "/upload" or "/uploadHash" |
| name | string{200} | + | Original file name of the uploaded file |

## PostCreationCommon
Common fields of both thread and reply creation requests

| Field | Type | Required | Description |
|---|---|:---:|---|
| image | [ImageRequest](#imagerequest) | - | Allocate a file together with the post |
| name | string{50} | - | Poster name and tripcode input |
| email | string{100} | - | Poster email |
| password | string{50} | + | Post password. Used for reclaiming a post after disconnection and preserving other limited post editing functionality, after closing a post. |

## ThreadCreationRequest

extends [PostCreationCommon](#postcreationcommon)

| Field | Type | Required | Description |
|---|---|:---:|---|
| subject | string{100} | + | thread subject |
| board | string{3} | + | board the thread will be inserted into |
| captcha | string | - | captcha response token |

## SpliceRequest
Mimics the behavior of JavaScript's [Array.prototype.splice](https://developer.mozilla.org/en/docs/Web/JavaScript/Reference/Global_Objects/Array/splice)
method.

| Field | Type | Required | Description |
|---|---|:---:|---|
| start | uint | + | Start position in the current line to begin splicing from |
| len | int | + | Length of the string that should be deleted, starting with at the start position. -1 has the special meaning of deleting everything from start position till the end of the line. |
| text | string | + | String to insert in the start position, after the "len" of the string has been deleted |

## SyncRequest

| Field | Type | Required | Description |
|---|---|:---:|---|
| thread | uint | + | ID of the thread to synchronise to . If synchronizing to a board page, set to `0`. |
| board | string{3} | + | Target board or parent board of  the thread |

## ReclaimRequest

| Field | Type | Required | Description |
|---|---|:---:|---|
| id | uint | + | ID of the post to reclaim |
| password | string{50} | + | Password of the target post |
