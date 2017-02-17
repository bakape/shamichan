[![GoDoc](https://godoc.org/github.com/bakape/meguca?status.svg)](https://godoc.org/github.com/bakape/meguca)
[![Build Status](https://travis-ci.org/bakape/meguca.svg)](https://travis-ci.org/bakape/meguca)

Platforms: Linux, OSX, Win64

License: GNU AGPL

##Features
* Posts and posting
    - Character by character post updates
    - Hovering quick reply for post authoring
    - Dice roll, coin flip and eightball commands
    - Desktop notifications, post highlighting and "(You)" on quote
    - Post link hover previews, including cross-thread
    - Inline post link expansion
    - Optional relative post timestamps
    - Image spoilering after closing a post
    - Non-temporal and recursive post linking
    - No posts per thread or threads per board limit
    - Forced anonymity display mode
    - Post hiding
    - Option to display only the last 100 posts in a thread
    - Optional automatic deletion of unused threads and boards
    - Automatic HTTP(S) and magnet URL linkification
    - Automatic quoting of selected text, when quoting a post
    - Live programming code tags with syntax highlighting
* Files and images
    - JPEG, PNG, APNG, WEBM, MP3, MP4, OGG, PDF, ZIP, 7Z, TAR.GZ and TAR.XZ
    supported
    - Transparent PNG and GIF thumbnails
    - Configurable size limits
    - Inbuilt reverse image search
    - No file is ever thumbnailed or stored twice, reducing server load and
    disk space usage
    - Any file already present on the server is "uploaded and thumbnailed"
    instantly
* Performance
    - Low memory and CPU usage
    - No frameworks and optimized code on both client and server
    - File upload processing written in C with GraphicsMagick and ffmpeg
    - Inbuilt custom LRU cache
* Client UI
    - Works with all modern and most outdated browsers (such as PaleMoon)
    - Read-only functionality preserved with JavaScript disabled
    - Scrolling compensation prevents post updates from moving the viewport
    - Inbuilt DOM update batching to reduce redraws
    - Multiple themes
    - Custom user-set backgrounds and CSS
    - Configurable keyboard shortcuts
    - Work mode aka Boss key
    - Customisable top banner board link list
    - Optional animated GIF thumbnails
    - Settings export/import to/from JSON file
* Board administration
    - User board creation and configuration panels
* Internationalization
    - Client almost entirely localized in multiple languages
    - More languages can be added by editing simple JSON files
* Miscellaneous
    - Documented public JSON API and WebSocket protocol
    - Optional R/a/dio Now Playing banner

##Runtime dependencies
* [PostgresSQL](https://www.postgresql.org/download/) >= 9.5

##Building from source
A reference list of commands can be found in `docs/installation.md`

###Build dependencies
* [Go](https://golang.org/doc/install) >=1.7 (for building server)
* [Node.js](https://nodejs.org) >=5.0 (for building client)
* GCC or Clang
* make
* pkg-config
* pthread
* ffmpeg >= 3.0 libraries (libavcodec, libavutil, libavformat) compiled with:
    * libvpx
    * libvorbis
    * libopus
    * libtheora
    * libx264
    * libmp3lame
* GraphicsMagick compiler with:
    * zlib
    * libpng
    * libjpeg
    * postscript
* git
* zip

###Linux and OSX
* Run `make`

###Windows
* Install [MSYS2](https://sourceforge.net/projects/msys2/)
* Open MSYS2 shell
* Install dependencies listed above with the `mingw-w64-x86_64-` prefix with
pacman
* Navigate to the meguca root directory
* Run `make`

##Setup
* See `./meguca help` for server operation
* Login into the "admin" account via the infinity symbol in the top banner with
the password "password"
* Change the default password
* Create a board from the administration panel
* Configure server from the administration panel

##Development
* See `/docs/` for more documentation
* `./meguca` or `./meguca debug` run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make update_deps` updates all dependencies
* `make clean` removes files from the previous compilation
* `make dist_clean` in addition to the above removes uploaded files and their
thumbnails

###Linux only
* make creates a Go workspace in the `.build` subdirectory. If you don't have a
proper Go workspace set up, you can simply `export GOPATH=$(pwd)/.build` to
temporarily assign `.build` as your Go workspace in the current shell. This will
allow you to use native go commands such as `go test` and `go build`.
