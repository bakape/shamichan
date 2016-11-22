Documentation of the WebSocket API. For commonly used JSON types in the API see
[common.md](common.md).

- The API only uses textual WebSocket frames for communication
- Only one message is transmitted per frame
- Each frame starts with two bytes with the ASCII-encoded message number. If the
message number is single digit, it must be padded with a leading zero.
- The first message after establishing a WebSocket connection to the server
must always be "synchronize".
- All complex payloads, such as JSON objects are JSON stringified

#Server to client

| Code | Name | Payload type | Description |
|:----:|--------------|---------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 0 | invalid | string | Convey an unrecoverable error. Only used on client protocol violations or server errors. The connection is terminated after writing this message. You should either fix your client implementation or report a server bug, if this message is encountered.  |
| 1 | insertThread | [ThreadCreationResponse](#threadcreationresponse) | Response to a thread creation request |
| 2 | insertPost | [Post](common.md#post) | Post insertion into the thread. The passed post may already exist and be rendered, in which case it is a possibly updated version of the post, that syncs the client's state to the update stream. In that case the client must rerender or deduplicate appropriately.  |
| 3 | append | [2]uint | Append a character to the current line of the post. The first array item is the ID of the target post. The seconds is a character encoded as UTF-8 character code. |
| 4 | backspace | uint | Remove one character from the end of the line of the post specified by ID. |
| 5 | splice | [SpliceMessage](#splicemessage) | Splice the current open line. Used for all text mutations, that are neither "append" or "backspace".  |
| 6 | closePost | uint | Close the post specified by ID. This message may be received for already closed posts, due to asynchronous nature of the eventual synchronisation algorithm.  |
| 7 | link | [LinkMessage](#linkmessage) | Insert a link into the specified post's link map. This message is always sent before the message to close an open line, so that any links are available, when the line is parsed.  |
| 8 | backlink | [LinkMessage](#linkmessage) | Add a backlink to the post specified by ID. |
| 9 | command | [CommandMessage](#commandmessage) | Append a command result to the specified post's array. Insert a link into the specified post's link map. This message is always sent before the message to close an open line, so that any command results are available, when the line is parsed.  |
| 10 | insertImage | [ImageMessage](#imagemessage) | Insert an image into an open post. |
| 11 | spoiler | uint | Spoiler the image of the post specified by ID |
| 30 | synchronize | map[uint][Post](common.md#post) | Response to a synchronization request. Contains a map of posts updated in the thread in the last 30 seconds. These are meant to bring the client up to sync with the update stream server-side. Consequently the client must ensure his existing post data is not more than 30 seconds old before synchronization. |
| 31 | reclaim | uint | Response to a request to reclaim a post lost after disconnecting from the server. 0 denotes success and the client is henceforth able to write to said post, as before the disconnect.1 denotes the post is unrecoverable.  |
| 41 | postID | uint | Returns the post ID of the client's freshly allocated post. A response to a post insertion request. |
| 42 | concat | * | Contains several null-byte concatenated messages. Used for limiting the rate of update frames sent from the server.  |

##ThreadCreationResponse

| Field | Type | Required | Description                                                                           |
|-------|------|----------|---------------------------------------------------------------------------------------|
| code  | uint | +        | Error code for the thread creation attempt. 0 for no error and 1 for invalid captcha. |
| id    | uint | +        | ID of the newly created thread                                                        |

##SpliceMessage
Mimics the behavior of JavaScript's [Array.prototype.splice](https://developer.mozilla.org/en/docs/Web/JavaScript/Reference/Global_Objects/Array/splice)
method.

| Field | Type | Required | Description |
|-------|--------|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| start | uint | + | Start position in the current line to begin splicing from |
| len | int | + | Length of the string that should be deleted, starting with at the start position. -1 has the special meaning of deleting everything from start position till the end of the line. |
| text | string | + | String to insert in the start position, after the "len" of the string has been deleted |

##LinkMessage

| Field | Type | Required | Description |
|-------|----------------------------------|----------|-------------------------------------------|
| id | uint | + | ID of the target post |
| links | [PostLinks](common.md#postlinks) | + | Links to be inserted into the target post |

##CommandMessage
extends [Command](common.md#command)

| Field | Type | Required | Description |
|-------|------|----------|-----------------------|
| id | uint | + | ID of the target post |

##ImageMessage
extends [Image](common.md#image)

| Field | Type | Required | Description |
|-------|------|----------|-----------------------|
| id | uint | + | ID of the target post |
