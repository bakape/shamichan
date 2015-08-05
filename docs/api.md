The read-only JSON API currently supports 4 modes of query.

####Format
| Query | URL | Returns |
|:-------:|:----------------------------:|:-------------------------------:|
| post | /api/post/${post number} | [post object](#post-object) |
| thread | /api/thread/${thread number}?last1=n* | array of [post objects](#post-object) |
| board | /api/board/${board name} | array of thread numbers in bump order |
| config | /api/config |client-side configuration object**|

\* `last?=n` is optional. `n` indicates the number of replies to retrieve from the thread bottom
\*\* Formated as `{config, hot}`. For an explanation of each exposed variable see `./config`

####Post object
| Key | Value | Optional | Exclusive | Example |
|:-------:|:----------------------------------------------------------------------------------------------------------------:|:--------:|:---------:|:------------------------------------------:|
| time | Unix timestamp in ms in the server's timezone | no | no | `1423578435043` |
| num | post number | no | no | `19` |
| op | thread OP post number | no | reply | `18` |
| board | post board | no | no | `"a"` |
| replies | array of post numbers in post order | no | OP | `["2", "3", "4"]` |
| replyctr | number of replies | no | OP | `10`|
| hctr | thread history counter; increments on each tread update | no | OP | `190` |
| omit | number of replies omittes with `last=n` | no | OP | `0` |
|image_omit | number of images omitted | no | OP | `12` |
| subject | thread subject | yes | OP | `"New Heights of Sodomy"` |
| name | poster name | yes | no | `"namefag"` |
| trip | poster tripcode | yes | no | `"!tripfag"` |
| email | poster email | yes | no | `"sage"` |
| mnemonic | poster IP in human readble format | no | mods | `"daadunu"`|
| auth | staff title; one of admin, moderator or janitor | yes | no | `"admin"` |
| body | post body | no | no | `">implying this is an example"` |
| image | [image object](#image-object) | yes  | no | [image object](#image-object) |
| dice | array of hash command result arrays | yes | no | `[[20, 0, 3], [2, 0, 2]]` |
| links | object of `${post number}: ${thread number}`<br>key-value pairs the current post is linking to | yes | no | `{"18":"17","27":"26"}` |
| backlinks | object of `${post number}: ${thread number}`<br>key-value pairs the current post is linked by | yes | no | `{"4":"1","5":"1"}` |

###Image object
| Key | Value | Optional | Exclusive | Example |
|:---:|:-----:|:--------:|:-------:|:--------:|
| src | image file name as hosted on the server | no | no| `"1423578439604.webm"` |
| thumb | image thumbnail name | yes | no | `"1423578439604.jpg"` |
| mid | high quality thumbnail name; presence depends on server configuration | yes | no | `"1423578439604.jpg"` |
| ext | image extension | no | no | `".webm"` |
| dims | image and thumbnail dimension array<br>`[${image width}, ${image height}, ${thumbnail width}, ${thumbnail height}]` | yes | no | `[640, 900, 89, 125]` |
| size | image file size in bytes | no | no | `105805` |
| MD5 | image MD5 hash | no | no | `"teVHnYA9Va1SRs2gPRIQ0A"` |
| SHA1 | image SHA1 hash | no | no | `"56df871ad268bb8b794bc61677bf3849e80db8f8"` |
| imgnm | original image name | no | no | `"illya dance.webm"` |
