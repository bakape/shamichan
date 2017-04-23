[![GoDoc](https://godoc.org/github.com/bakape/meguca?status.svg)](https://godoc.org/github.com/bakape/meguca)
[![Build Status](https://travis-ci.org/bakape/meguca.svg)](https://travis-ci.org/bakape/meguca)

Platforms: Linux, OSX, Win64

License: GNU AGPL

## Features
<details>
    <summary>Posts and posting</summary>
    <ul>
        <li>Character by character post updates</li>
        <li>Hovering quick reply for post authoring</li>
        <li>Dice roll, coin flip and eightball commands</li>
        <li>Desktop notifications  and "(You)" links on quote</li>
        <li>Post link hover previews, including cross-thread</li>
        <li>Inline post link expansion</li>
        <li>Optional relative post timestamps</li>
        <li>Non-temporal and recursive post linking</li>
        <li>No posts per thread or threads per board limit</li>
        <li>Forced anonymity display mode</li>
        <li>Post hiding</li>
        <li>Option to display only the last 100 posts in a thread</li>
        <li>Optional automatic deletion of unused threads and boards</li>
        <li>Automatic URL linkification</li>
        <li>Automatic intelligent quoting of selected text, when quoting a post</li>
        <li>Live programming code tags with syntax highlighting</li>
        <li>Automatic open post recovery after a disconnect</li>
        <li>Toggleable non-live post creation</li>
        <li>Keyboard post navigation</li>
    </ul>
</details>
<details>
    <summary>Files and images</summary>
    <ul>
        <li>
            JPEG, PNG, APNG, WEBM, MP3, MP4, OGG, PDF, ZIP, 7Z, TAR.GZ and
            TAR.XZ are supported
        </li>
        <li>Transparent PNG and GIF thumbnails</li>
        <li>Configurable size limits</li>
        <li>Inbuilt reverse image search</li>
        <li>
            No file is ever thumbnailed or stored twice, reducing server load
            and disk space usage
        </li>
        <li>Any file already present on the server is "uploaded and thumbnailed"</li>
        <li>Gallery mode</li>
        instantly
    </ul>
</details>
<details>
    <summary>Performance</summary>
    <ul>
        <li>Low memory and CPU usage</li>
        <li>No frameworks and optimized code on both client and server</li>
        <li>File upload processing written in C with GraphicsMagick and ffmpeg</li>
        <li>Inbuilt custom LRU cache</li>
    </ul>
</details>
<details>
    <summary>Client UI</summary>
    <ul>
        <li>Works with all modern and most outdated browsers (such as PaleMoon)</li>
        <li>Works with JavaScript disabled browsers</li>
        <li>Read-only functionality preserved with JavaScript disabled</li>
        <li>Multiple themes</li>
        <li>Custom user-set backgrounds and CSS</li>
        <li>Mascots</li>
        <li>Configurable keyboard shortcuts</li>
        <li>Work mode aka Boss key</li>
        <li>Customisable top banner board link list</li>
        <li>Optional animated GIF thumbnails</li>
        <li>Settings export/import to/from JSON file</li>
    </ul>
</details>
<details>
    <summary>Board administration</summary>
    <ul>
        <li>Support for both centralized and 8chan-style board ownership</li>
        <li>Global admin -> users notification system</li>
        <li>User board creation and configuration panels</li>
        <li>4 tier staff system</li>
        <li>Board-level and global bans</li>
        <li>Transparent post deletion</li>
    </ul>
</details>
<details>
    <summary>Internationalization</summary>
    <ul>
        <li>Client almost entirely localized in multiple languages</li>
        <li>More languages can be added by editing simple JSON files</li>
    </ul>
</details>
<details>
    <summary>Miscellaneous</summary>
    <ul>
        <li>Documented public JSON API</li>
        <li>Optional R/a/dio Now Playing banner</li>
        <li>Synchronized time counters (for group watching sessions and such)</li>
        <li>Thread-level connected unique IP counter</li>
        <li>Internal captcha system</li>
    </ul>
</details>

## Runtime dependencies
* [PostgresSQL](https://www.postgresql.org/download/) >= 9.5

## Building from source
A reference list of commands can be found in `./docs/installation.md`

### Build dependencies
* [Go](https://golang.org/doc/install) >=1.8 (for building server)
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
