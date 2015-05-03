Currently the read-only JSON API is hosted on a separate webserver on its own port (8002 by default), thus it is recommended to reverse proxy it as shown in the sample nginx configuration file. The API currently supports 5 modes of query.

####Format
| Query | URL | Returns |
|:-------:|:----------------------------:|:-------------------------------:|
| post | /api/post/${post number or comma-separated list of post numbers}* | array of post objects |
| thread | /api/thread/${thread number or comma-separated list of thread numbers}* | array of arrays of post objects |
| board | /api/board/${board name} | array of arrays of post objects |
| catalog | /api/catalog/${board name} | array of post objects |
| config | /api/config |client-side configuration object* |

\* When passing a single number, the array will contain only one post object. Passing multiple like `1,6,5`, will return an array if these posts in the specified order. If even one post is not accessible, the entire request will 404.
\*\* Formated as `{config, hot}`. For an explanation of each exposed variable see ./config

####Values
| Key | Value | Optional | Exclusive | Example |
|:-------:|:----------------------------------------------------------------------------------------------------------------:|:--------:|:---------:|:------------------------------------------:|
| time | Unix time of post in the server's timezone | no | no | "1423578435043" |
| num* | post number | no | no | "19" |
| op | thread OP post number | no | reply | "18" |
| board* | post board | no | no | "a" |
| replies | number of replies in a thread | no | OP | "3541" |
| src | image file name as hosted on the server | yes | no | "1423578439604.webm" |
| thumb | image thumbnail name | yes | no | "1423578439604.jpg" |
| mid | high quality thumbnail name; presence depends on server configuration | yes | no | "1423578439604.jpg" |
| ext* | image extension | yes | no | ".webm" |
| dims | comma-separated list of image dimension; "${image width},${image height},${thumbnail width},${thumbnail height}" | yes | no | "640,900,89,125" |
| size | image file size in bytes | yes | no | "105805" |
| MD5 | image MD5 hash | yes | no | "teVHnYA9Va1SRs2gPRIQ0A" |
| SHA1 | image SHA1 hash | yes | no | "56df871ad268bb8b794bc61677bf3849e80db8f8" |
| imgnm | original image name | yes | no | "illya dance.webm" |
| subject | thread subject | yes | OP | "New Heights of Sodomy" |
| name | poster name | yes | no | "namefag" |
| trip | poster tripcode | yes | no | "!tripfag" |
| email | poster email | yes | no | "sage" |
| body | post test body | no | no | ">implying this is an example" |
| links | object containing "${post number}: ${thread number}" key-value pairs the current post is linking to | yes | no | {"18":"17","27":"26"} |

\* Will not be defined, when reading older posts created prior to 068b99941d7d60a0524d8252b814fd0053a0da1d (v0.10.0)

