[![GoDoc](https://godoc.org/github.com/bakape/meguca?status.svg)](https://godoc.org/github.com/bakape/meguca)
[![Build Status](https://travis-ci.org/bakape/meguca.svg)](https://travis-ci.org/bakape/meguca)
[![Dependency Status](https://david-dm.org/bakape/meguca.svg)](https://david-dm.org/bakape/meguca)

Platforms: Linux, OSX, Win64

License: GNU AGPL

##Features
* Posts and posting
    - Character by character post updates
    - Can edit entire line while writing
    - Hovering quick reply for post authoring
    - Dice roll, coin flip and eightball commands
    - Desktop notifications, post highlighting and "(You)" on quote
    - Post link hover previews, including cross-thread
    - Optional relative post timestamps
    - Image spoilering after closing a post
    - Non-temporal and recursive post linking
    - No posts per thread or threads per board limit
    - Forced anonymity display mode
    - Post hiding
    - Option to display only the last 50 posts in a thread
    - Optional automatic deletion of unused threads and boards
    - Automatic HTTP(S) and magnet URL linkification
* Files and images
    - JPEG, PNG, APNG, WEBM, MP3, MP4, OGG, ZIP, 7Z, TAR.GZ and TAR.XZ supported
    - Transparent PNG and GIF thumbnails
    - Configurable size limits
    - Inbuilt reverse image search
    - No file is ever thumbnailed or stored twice, reducing server load and
    disk space usage
    - Any file already present on the server is "uploaded and thumbnailed"
    instantly
* Performance
    - Low memory and CPU usage
    - High vertical and easy horizontal scaling
    - No frameworks and optimized code on both client and server
    - Fast video and audio processing through C bindings to ffmpeg
    - On-demand lazy client module loading and rendering
* Client UI
    - Scrolling compensation prevents post updates from moving the viewport
    - Inbuilt DOM update batching to reduce redraws
    - Multiple themes
    - Custom user-uploaded backgrounds
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
    - Optional R/a/dio Now Playing banner
    - Public JSON API

##Runtime dependencies
* [RethinkDB](https://rethinkdb.com/docs/install/).
On Linux RethinkDB does not enable a configuration file by default. If you don't
want to configure anything, just copy `/etc/rethinkdb/default.conf.sample` into
`/etc/rethinkdb/instances.d/instance1.conf`.

##Compiled Windows Binaries
Compiled binary release archives for windows/x86_64 are downloadable from the
[release](https://github.com/bakape/meguca/releases) page.

##Setup
* See `./meguca help` for server operation
* Login into the "admin" account via the infinity symbol in the top banner with
the password "password"
* Change the default password
* Create a board from the administration panel
* Configure server from the administration panel

##Building from source

###All Platforms
* Install
	* [Go](https://golang.org/doc/install) >=1.7
	* [Node.js](https://nodejs.org) >=5.0 (for building the client)

###Linux and OSX
* Install
    * GCC or Clang
    * make
    * ffmpeg >= 3.0 development libraries (libavcodec, libavutil, libavformat,
    libswscale) compiled with
        * libvpx
        * libvorbis
        * libopus
        * libtheora
        * libx264
        * libmp3lame
    * git
    * zip
* Run `make`

###Windows
* Install [MSYS2](https://sourceforge.net/projects/msys2/)
* Open MSYS2 shell
* Install with pacman
    * mingw-w64-x86_64-ffmpeg
    * mingw-w64-x86_64-gcc
    * mingw-w64-x86_64-pkg-config
    * git
    * make
    * zip
* Navigate to the meguca root directory
* Run `make`

##Development
* See `/docs/` for more documentation
* `./meguca` or `./meguca debug` run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make update` updates all dependencies and fetches new project sources from
the master branch
* `make clean` removes files from the previous compilation
* `make dist_clean` in addition to the above removes uploaded files and their
thumbnails

###Linux only
* make creates a Go workspace in the `.build` subdirectory. If you don't have a
proper Go workspace set up, you can simply `export GOPATH=$(pwd)/.build` to
temporarily assign `.build` as your Go workspace in the current shell. This will
allow you to use native go commands such as `go test` and `go build`.
