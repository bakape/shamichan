# Change Log
All notable changes to this project will be documented in this file.
Project uses [Semantic Versioning](http://semver.org/)

##v3.2.0 - 2016-12-20
* Fix
	- Inline post border visibility on all themes
* Add
	- Reference installation commands
	- Highlight reverse reference links in inlined posts

##v3.1.0 - 2016-12-19
* Change
	- Improved page loading and JSON/HTML generation Speed
	- Improved file thumbnailing speed
	- Added "thumbExt" field to post JSON. Seed docs.
	- Choose between JPEG and PNG thumbnails based on wether the image has
	transparency
* Fix
	- Cross-thread post hover previews
	- Paste handling in the post input form
* Remove
	- Lossy PNG thumbnail compression
* Add
	- Dependency for GraphicsMagick and pthreads
	- PDF upload support
	- User-set custom CSS option
	- Highlight target post on link hover preview
	- Automatic quoting of multiline pastes, if line starts with `>`
	- Automatic quoting of selected text, when quoting a post
	- Inline post link expansion

##v3.0.0 - 2016-12-12
* BREAKING:
	- Changed CLI flags to shorter one letter variants
	- Removed `--origin` CLI flag
	- Restructured JSON API and websocket protocol. See `docs/` for changes.
* Change
	- More efficient hybrid server- and client-side rendering
	- More compact [Rules] and [Notice] widgets
	- Revert valid Last N posts parameters to 5 and 100
* Add
	- inumi theme
	- Polish language pack (partial)
	- HTML, JSON and DB query LRU cache
	- [Expand] button to catalog threads without images
	- Read-only site usability with JavaScript disabled
	- Full site functionality with outdated browsers like PaleMoon
* Remove
	- Client origin checking
	- Email field for post creation
	- Image hats
	- Image link shortening in figcaption
* Fix
	- Relative timestamps not toggling off without refresh
	- Downloading archives on thumbnail click in FireFox

##v2.7.1 - 2016-11-30
* Fix
	- Backspace corrupting posts, when tab hidden
	- Catalog sorting and searching

##v2.7.0 - 2016-11-29
* Fix
	- Not compiling due to TypeScript type checker regression
	- Images expanding in post previews
	- Uncaught synchronization errors on connectivity loss
	- Boards not deleting themselves after expiry
	- Reply form taking too much viewport height
	- CloudFlare IP forwarding Nginx sample configs
	- Unshown posts inserting to thread bottom in Last N posts display mode
* Add
	- Alt+up keybind to navigate to the upper board level
	- Margin to embedded content
	- ZIP, 7Zip, TAR.GZ and TAR.XZ archive upload support
	- Websocket protocol documentation
	- "Inumi" dark Material Design theme
* Change
	- Make images optional for thread creation
* Remove
	- Websocket connection origin restrictions
	- "--origin" CLI parameter

##v2.6.1 - 2016-11-02
* Fix
	- Splicing of multibyte character unicode strings

##v2.6.0 - 2016-11-02
* Add
	- OGG and MP4 upload support
	- Preview new post credentials before allocation
	- Configurable old board and thread pruning
	- Link to unload to the last 50 posts at thread bottom
	- Floating quick reply
	- Box shadows to all floating UI elements, except image previews
	- Desktop notifications on quoting the user's posts
* Change
	- Speed up upload and thumbnailing of files already present on the server
	- Fix configurable Last N post display number to 50 posts
	- Optimised large thread overall performance and responsiveness
* Fix
	- Omitted post and image span wrapping
	- Text spoiler display honoring board spoiler configuration
	- Forced anonymity setting
	- Spaces not displaying after text spoilers
	- SVG link icon colours
	- Middle clicking on image search links
	- Missing content after resuming from standby or a background tab on mobile

##v2.5.1 - 2016-10-23
* Fix
	- Board configuration live reloading

##v2.5.0 - 2016-10-22
* Fix
	- Existing board updates on board creation
	- Don't lock to bottom, when navigating between threads
* Add
	- Cross-thread post previews
	- Underline reverse links in post hover previews

##v2.4.1 - 2016-10-21
* Fix
	- Page not loading, when there are no boards created

##v2.4.0 - 2016-10-18
* Add
	- Youtube, SoundCloud and Vimeo embeds
	- Optional notice for mature content
	- Ability to continue posts during connection loss
* Fix
	- Large post preview positioning in FireFox
	- Firefox jumping to thread top on post link click
	- Inconsistencies in scrolling to posts on link clicks
	- Possible crashes on WebM and MP3 processing
* Change
	- Hide file upload controls while uploading
	- Scroll to bottom on post creation

##v2.3.0 - 2016-10-08
* Warning
	- An automatic database upgrade will be performed. Make sure only one
	instance of meguca is running at that time. It is also recommended to back
	up the database before upgrading.
* Add
	- Highlighting of posts quoting you
	- MP3 uploads even without cover art
	- Board rule display widget
	- Board notice display widget
	- Catalog thread sorting and filtering
	- Manual catalog refreshing
	- Automatic catalog refreshing, when tab is hidden
	- Automatically delete boards that have not had any posts for 7 days
	- Make rule for updating project and dependencies
	- Ukrainian and Slovak language packs
	- Ability to spoiler images after allocation
* Change
	- Navigatable board selection sidebar
	- Hide successive empty newlines
	- Batch concatenated live updates in 0.2 second intervals
	- Don't log abnormal websocket closure errors
* Remove
	- [New Thread] button from board page bottom
	- Legacy deployment script. Superseded by binary releases.
	- ES5 support. All browsers are now required to support ES6 and are notified
	to update, if feature detection fails.
* Fix
	- #flip always returning false
	- Drastically improved live update CPU and memory usage. Updates in will no
	longer slow down as a thread gets larger.
	- File upload attempt handling on text-only boards
	- Etag generation on upload assets and JSON

##v2.2.0-beta - 2016-09-02
* Add
	- Configurable board navigation panel
	- Board configuration WebUI
	- Optional Solve Media captcha integration
	- Thread and post creation and basic updating
* Change
	- Required Go version to 1.7
	- Move origin configuration to server flag
	- Restrict board names to [a-z0-9]{1,3}
	- Remove minimum length requirement for passwords
* Fix
	- Client HTTPS detection
	- SystemJS not loading in Palemoon
	- Board list retrieval with no boards created
	- Captcha reloading in login/registration forms

##v2.1.0-alpha - 2016-07-04
* License
	- Relicense under GNU AGPL
* Add
	- HTTP and database connection flags. See `./meguca help`.
	- Server configuration WebUI
	- Basic account management for staff
	- User-creatable boards
	- Global "admin" account
* Change
	- Highlight all banner panel toggles, until clicked
	- Descriptive text error pages instead of graphical ones
	- Hard code /all/ as default board
	- Make spoilers board-specific
* Fix
	- Duplicate header writing, when using Chrome dev tools
* Remove
	- Staff board
	- Pseudo boards
	- `config/config.json` configuration file

##v2.0.0-alpha - 2016-06-21
Start tracking v2 progress in versions for easier debugging
	- Rewriting server in Go
	- Rewriting client in TypeScript
	- Switched DBMS to RethinkDB

##v1.9.6 - 2016-08-11
* Add
	- Turkish language pack

##v1.9.5 - 2016-07-12
* Fix
	- Outdated desustorage URL

##v1.9.4 - 2016-06-21
* Fix
	- Server crashing, when launched from init script on Debian

##v1.9.3 - 2016-06-11
* Fix
	- Don't auto expand files with audio and PDF

##v1.9.2 - 2016-05-27
* Fix
	- Server crashes on catalog pages

##v1.9.1 - 2016-04-29
* Fix
	- Spontaneous server crashes

##v1.9.0 - 2016-04-29
* Change
	- Replace homebrew web app install prompt with Android Chrome install banner

##v1.8.3 - 2016-04-17
* Change
	- Restore web manifest install prompt on Android Chrome/FF

##v1.8.2 - 2016-04-11
* Fix
	- Revert bugged SockJS version change

##v1.8.1 - 2016-04-05
* Fix
	- Websocket heartbeat timeout crashes

##v1.8.0 - 2016-04-03
* Add
	- Material Design theme
* Fix
	- Syncing of mobile devices

##v1.7.5 - 2016-04-01
* Fix
	- Not being able to connect to websocket in some cases

##v1.7.4 - 2016-03-28
* Fix
	- Post hover anonymisation

##v1.7.3 - 2016-02-12
* Change
	- Lock down current stable, so we can move v2 into master

##v1.7.2 - 2015-12-27
* Fix
	- Server crashing on no thumbnail catalog OP renders
	- Image hat z-index
* Add
	- Video controls to all WebM
    - Release publishing helper script

##v1.7.1 - 2015-12-10
* Fix
	- r/a/dio banner not updating after prolonged operation
* Remove
	- Dedicated saucenao.com SSL bypass
* Change
	- Only DJs allowed to wear names with forced anon on

##v1.7.0 - 2015-12-05
* Remove
	- Drop shadow from image hover previews
* Add
	- DJ staff class with ability to scan for song requests

##v1.6.2 - 2015-11-21
* Fix
	- PDF thumbnailing timeout

##v1.6.1 - 2015-11-15
* Fix
	- Post links in report emails

##v1.6.0 - 2015-11-13
* Fix
	- Page not loading on outdated Chrome versions
* Add
	- Restore Vagrant support

##v1.5.1 - 2015-11-10
* Fix
	- "hide" image thumbnail mode

##v1.5.0 - 2015-11-08
* Add
	- Work mode aka Boss key

##v1.4.0 - 2015-11-08
* Add
	- Optional site frontpage to serve on '/'
* Changed
	- Upgrade breaking dependencies

##v1.3.3 - 2015-11-06
* Fix
	- Writing of multiple dice in one fragment

##v1.3.2 - 2015-11-06
* Fixed
	- Crash on failing to read moderation array

##v1.3.1 - 2015-10-31
* Fixed
	- New post multiplication in Expand All image mode
	- Crashes related to database dice spec migration
	- Connection getting stuck on "Syncing"
* Changed
	- Hide loading indicator on fetch failure
* Added
	- Babel.js transformer check for node.js version >=5
	- Switch from archive.moe image search to desustorage.org

##v1.3.0 - 2015-10-20
* Fixed
	- Crash on parsing dice server-side in some cases
* Added
	- Option to hide some boards from navigation bar

##v1.2.7 - 2015-10-11
* Fixed
	- "Works best with" appearing on Chromium browsers
	- Skewed clocks resulting in post showing in the future

##v1.2.6 - 2015-10-10
* Fixed
	- Deploy script iptables persistence after reboot
	- Soundcloud embedding under HTTPS
* Changed
	- JS injections now execute on all thread's and board's sections, articles
    and post forms

##v1.2.5 - 2015-10-04
* Fixed
	- Post reporting

##v1.2.4 2015-10-03
* Fixed
	- Moderation spoiler live update

##v1.2.3 2015-10-03
* Fixed
	- New post keyboard shortcut

##v1.2.2 - 2015-10-02
* Changed
	- Refactored page scrolling on change compensation
	- Default image hover preview and relative timestamp settings

##v1.2.1 - 2015-09-30
* Fixed
	- Staff logging in

##v1.2.0 - 2015-09-30
* Added
	- Ability to select redis database to use
	- Automatic setup script for Ubuntu Trusty
* Removed
	- HTML caching

##v1.1.1 - 2015-09-23
* Fixed
	- Post menu background artefact
	- Missing post menu button on console theme
	- Reload looping, when behind CDN

##v1.1.0 - 2015-09-22
* Added
	- 40x and 50x page rendering on requests
	- DOM level 4 polyfill
* Changed
	- Modal design in console theme
* Fixed
	- Missing top <hr> on thread pages
	- Missing icons in "Works best with" message

##v1.0.0 - 2015-09-20
* Added
	- UTC Clock to the schedule
	- Options export and import to/from file
	- Option to anonymise all posters
	- Selectable language packs
	- Image banners
	- Live updates in post hover previews
	- Support for hosting static assets on a subdomain
	- Ocean theme
	- Full ETag support
	- Debug mode forcing client-side with `debug=true` query string
	- Compatibility bundle for older, hipster and outright retarded browsers
	- Scroll to post after contracting images taller than the viewport
	- `scripts/send` for pushing arbitrary messages to all client from the
    server's shell
	- Box shadow to upper layer elements
	- Loading indicator
	- Staff board to board navigation
	- Janitor staff class
	- Link hover colour to glass theme
	- Basic benchmark script
	- Option to disable moderation on specific boards
	- Moderation taken indicators for staff
	- Moderation log for staff
	- Internal production error logging
	- node.js v3 support
	- `--debug` flag for forcing debug mode
	- Database migration script from v0 or vanilla doushio
	- Reason field for bans
	- <noscript> header
	- Panel listing active
	- Full client-facing localization
	- Operational dev guide
	- Separate mobile HTML templates
	- Uncommitted text colour to glass theme
	- Supported browser indicator
	- Configurable custom `>>>/${link}/` targets
	- Optional global server-side anonymisation
* Changed
	- Client rewritten mostly form scratch
	- Meguca client compilation, install and upgrade procedures. See README.md
	- Pastebin embed height to 0.65 of viewport
	- Client to single page application
	- Faster builder.js client recompilation on change
	- Upgraded dependencies, including minimal node.js version
	- Google image search enabled by default
	- Post menu only disappears on click either inside or outside the menu
	- Render catalog server-side
	- JSON API spec
	- Metric ton of performance improvements
	- Switched to 4chan-like thread expiry model with page limits and autosaging
	- Persist `sage` in the email field
	- Clicking on catalog images no longer opens new tab
	- Cryptographically secure mnemonics
	- Always render inter-board navigation
	- Updated JSON API post spec
	- Restyle moderation selection checkbox
	- Hide mnemonics from janitors
	- Persistent JS script injections
	- Don't preload all spoiler panes on postform render
	- More efficient static asset caching
* Fixed
	- Delay before r/a/dio banner appearing, when enabled
	- Opus WebM parsing
	- Random images generating small thumbnails
	- Vagrant provisioning on OSX
	- Disconnect favicon 404, if served from subdomain
	- Navigating with inter-board post links
	- Thread hiding
	- Memory leak, if repeatedly clicking on UTC clock
	- Following a locked page bottom on post text input
	- Ctrl clicking post links
	- DOM bumping on reply shift on board pages
	- (You) disappearing on page refresh
	- Option exporting on Firefox
	- Image hover previews not respecting aspect ration on Firefox
	- Opening new tab when clicking audio controls in Firefox
	- Dangling apostrophe in tripcode
	- WebM expansion on Chrome for Android
	- Overflow of admin notifications
	- Purging hidden post list
	- Outline around [Return] after clicking [Bottom]
	- Deliberate server crashing, by sending certain websocket messages
	- Incorrect process exit codes on termination
* Removed
	- Archive board
	- Vagrant support
	- Board-specific default themes
	- Noko email field option
	- Outdated file map
	- Graveyard board
	- Fun threads
	- Changelog banner icon
	- Post focusing menu option
	- Board curfews
	- Imager daemon stub
	- Dynamic post unloading (temporarily, until reimplementation)
* Breaking
	- Dice storage in old threads
	- Backlink generation in old threads
	- Changed server entry point. Use `npm start` to start the server.
	- Removed thread tagging. See `docs/migration.md`.

##v0.11.3 - 2015-03-31
* Fixed
	- Server crash, when thread expiry is not defined for all boards

##v0.11.2 - 2015-03-19
* Fixed
	- "Clear hidden" button now only renders on the General tab of options
	- No word auto completion in the blockquote on mobile
	- Auto quoting the header, when selected

##v0.11.0 - 2015-03-15
* Added
	- PDF uploads
	- Automatic selected tex quoting
	- `#q` hash command for printing the r/a/dio song queue
	- Monit configuration file samples
	- Automatic linkification of >>>/board/ URLs
	- Pastebin link embedding
	- MP3 uploads (currently MP3s must have cover art)
	- Keybind for expanding all images
	- Automatic CSS file versioning and minification
	- Changelog icon to the banner
* Fixed
	- Administrator panel rendering on some configurations
	- Silent desynchronization on mobile
	- Syncwatch imprecision
	- Memory leak in syncwatch
	- `Could not get response` error, when uploading with drag&drop
	- Disappearing replies on mobile
* Changed
	- `npm install` now copies configuration files from examples on first run
	- Image hover previews now use the more responsive velocity.js library
	- Moved r/a/dio API polling server-side. Now passed to the clients through
    push notifications
	- Patched upkeep/radio.js to use push notifications
	- Made `#pyu` toggleable in the config.js
	- Reduced the banner''s screen footprint
	- Moved the separate archive daemon into a toggleable server module
	- Log errors to meguca's root directory
	- Updated init script sample
	- Notification text colour to red for easier noticeability
	- `make clean` to also delete compiled JS and CSS files
	- Vagrant now supplies all of the dependencies, including optional. Be sure
    to run `vagrant provision` for existing VMs
* Removed
	- Deployment script
	- Default filter from the sample config.js
	- Underline formatting for links
	- Keyboard shortcuts options tab for mobile
	- EXIF deletion script
	- Autocompletion of text in the blockquote

##v0.10.0 - 2015-02-24
* Added
	- Vagrant support
	- Readonly JSON API
	- Store more values in post hashes (features that depend on these will not
    work on threads created prior to 0.10.0)
	- Thread catalog
	- Option to disable staff IP tagging
	- Adding custom URLs to board navigation
	- youtu.be link embedding support
* Fixed
	- io.js and node >= 0.12.0 compilation on OSX
	- websocket hangups on node server disconnect
	- Caching issues on Android
* Changed
	- Expanded and reorganized documentation

##v0.9.0 - 2015-02-17
* Added
	- Chrome for Android home screen webapp support
	- Option to unlock from page bottom on tab visibility loss
	- Tabbed UI for options menu
* Fixed
	- Administrator notifications on smaller screens
	- ffmpeg RAM usage (still need plenty of RAM to thumbnail large WebM)
* Removed
	- Anon hours
* Changed
	- APNG detection and image duplicate comparison now use native C++ node addons
	- Dynamic post unloading now off by default
	- Syncwatch now always on
	- All threads now have both [Expand] and [Last n] links
	- Now using [io.js](https://iojs.org) instead of node.js. At the moment of
    writing should still be compatible with node.js v0.12.x. When upgrading be
    sure to run `make clean; npm update; npm install` from meguca's root
    directory.

##v0.8.0 - 2015-01-30
*  Changed
	- Acquire most dependencies through npm
	- Moved various settings from config.js to hot.js
	- Duplicate image detection similar to findimagedupes.pl
	- Replaced homebrew client builder with gulp.js
    (please run `# npm install -g gulp` and rerun `npm install`)
*  Fixed
	- Syncwatch imprecision
	- Report Recaptcha from HTTPS pages
	- Illya dance + glass theme distortion on webkit browsers
*  Removed
	- Posted From signatures
*  Added
	- Button for Fun Thread dispatching
	- Client configuration push updates
	- A fun in-model hook
	- Clicking on desktop notifications focuses reply tab and post
	- Displayed desktop notification memory
	- Favicons indicate thread status
	- Live administrator announcements
	- Text spoiler keybind

##v0.7.0 - 2015-01-15
*  Added
	- Ban purge script
	- mega.co.nz backup script
	- Live-toggleable image-related settings
	- Staff session purge script
	- Expand all images button
*  Removed
	- Large thumbnail mode
	- Japanese text-to-speech
	- Composite spoilers
	- WebM audio spoilers
*  Changed
	- Report emails activate moderation multiselection
	- More complete post model extraction

##v0.6.0 - 2014-12-26
*  Added
	- [Top] link in all browsers
	- Push websocket messages
	- Online user counter
*  Changed
	- Mods can now ban

##v0.5.1 - 2014-12-22
*  Added
	- Hot-reloadable word filter
	- Ability to render minimal page content (w/o scripts and CSS)
	- Dynamic post unloading
*  Removed
	- Legacy wordfilter.js
*  Changed
	- Last N query string syntax
	- Thread title sync to page title
	- Disabled some broken capability on mobile browsers
*  Fixed
	- Faulty ban timeout assignment

##v0.4.0 - 2014-12-3
*  Info
	- Started versioning separately from
    [doushio](https://github.com/lalcmellkmal/doushio)
