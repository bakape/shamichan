[![GoDoc](https://godoc.org/github.com/bakape/meguca?status.svg)](https://godoc.org/github.com/bakape/meguca)

# meguca
real-time anonymous imageboard focused on high performance, free speech and transparent moderation

Platforms: Linux, OSX, Win64

License: GNU AGPL

## Features

<details><summary>Posts and posting</summary>

- Character by character post updates
- Hovering quick reply for post authoring
- Dice roll, coin flip and eightball commands
- Desktop notifications  and "(You)" links on quote
- Post link hover previews, including cross-thread
- Inline post link expansion
- Optional relative post timestamps
- Non-temporal and recursive post linking
- No posts per thread or threads per board limit
- Forced anonymity display mode
- Post hiding
- Option to display only the last 100 posts in a thread
- Optional automatic deletion of unused threads and boards
- Automatic URL linkification
- Automatic intelligent quoting of selected text, when quoting a post
- Live programming code tags with syntax highlighting
- Automatic open post recovery after a disconnect
- Toggleable non-live post creation
- Keyboard post navigation
- Explicitly visible sage
- Responsive seen post detection
- Score-based antispam system
- Optional recursive post hiding

</details>

<details><summary>Files and images</summary>

- JPEG, PNG, APNG, WEBM, MP3, FLAC, MP4, OGG, PDF, ZIP, 7Z, TAR.GZ, TAR.XZ, TXT
are supported
- Transparent PNG and GIF thumbnails
- Configurable size limits
- Inbuilt reverse image search
- No file is ever thumbnailed or stored twice, reducing server load and disk space usage
- Any file already present on the server is "uploaded and thumbnailed" instantly
- Title metadata extraction
- Gallery mode

</details>

<details><summary>Performance</summary>

- Low memory and CPU usage
- No frameworks and optimized code on both client and server
- File upload processing written in C with GraphicsMagick and ffmpeg
- Inbuilt custom multi-level LRU cache

</details>

<details><summary>Client UI</summary>

- Works with all modern and most outdated browsers (such as PaleMoon)
- Works with JavaScript disabled browsers
- Multiple themes
- Custom user-set backgrounds and CSS
- Mascots
- Configurable keyboard shortcuts
- Work mode aka Boss key
- Customisable top banner board link list
- Optional animated GIF thumbnails
- Settings export/import to/from JSON file

</details>

<details><summary>Board administration/moderation</summary>

- Support for both centralized and 8chan-style board ownership
- Global admin -> users notification system
- User board creation and configuration panels
- 4 tier staff system
- Board-level and global bans
- Transparent post deletion
- Viewing of all post made by same IP
- Deleting all posts with same IP
- Option to disable search indexing on board
- Sticky threads
- Public ban list
- Public moderation log
- Mod image spoilering
- Image banners
- Custom per-board loading image
- Default board theme setting
- Optional poster country flag display
- Option to hide NSFW boards from /all/

</details>

<details><summary>Internationalization</summary>

- Client almost entirely localized in multiple languages
- More languages can be added by editing simple JSON files

</details>

<details><summary>Miscellaneous</summary>

- Documented public JSON API
- Optional R/a/dio Now Playing banner
- Synchronized time counters (for group watching sessions and such)
- Thread-level connected unique IP counter
- Internal captcha system

</details>

## Runtime dependencies
* [PostgresSQL](https://www.postgresql.org/download/) >= 9.5

## Building from source
A reference list of commands can be found in `./docs/installation.md`

### Build dependencies
* [Go](https://golang.org/doc/install) >=1.9 (for building server)
* [Node.js](https://nodejs.org) >=5.0 (for building client)
* GCC or Clang
* make
* pkg-config
* pthread
* ffmpeg >= 3.1 libraries (libavcodec, libavutil, libavformat, libswscale)
compiled with:
    * libvpx
    * libvorbis
    * libopus
    * libtheora
    * libx264
    * libmp3lame
* GraphicsMagick compiled with:
    * zlib
    * libpng
    * libjpeg
    * postscript
* git
* zip

### Linux and OSX
* Run `make`

### Windows
* Install [MSYS2](https://sourceforge.net/projects/msys2/)
* Open MSYS2 shell
* Install dependencies listed above with the `mingw-w64-x86_64-` prefix with
pacman
* Navigate to the meguca root directory
* Run `make`

## Setup
* See `./meguca help` for server operation
* Login into the "admin" account via the infinity symbol in the top banner with
the password "password"
* Change the default password
* Create a board from the administration panel
* Configure server from the administration panel
* To enable country flags on posts download and place `GeoLite2-Country.mmdb`
into the root directory

## Development
* See `./docs` for more documentation
* `./meguca` or `./meguca debug` run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make update_deps` updates all dependencies
* `make clean` removes files from the previous compilation
* `make dist_clean` in addition to the above removes uploaded files and their
thumbnails
* To enable using Go tools in the project add the absolute path of `./go` to
your `$GOPATH` environment variable
* For developing the new C++ client:
   - Run `git submodule init`
   - Install [Emscripten](http://kripken.github.io/emscripten-site/docs/getting_started/downloads.html)
   - Use `DEBUG=1 make wasm` and `make clean_wasm` to compile the C++ client and clean build directories
   - To use the C++ client for meguca add the `?wasm=true` query string to the end of the URL
