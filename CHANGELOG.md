# Change Log
All notable changes to this project will be documented in this file.
Project uses [Semantic Versioning](http://semver.org/)

##1.0.0 - 2015-09-20
###Added
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
- Compatability bundle for older, hipster and outright retarded browsers
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
- Full client-facing localisation
- Operational dev guide
- Separate mobile HTML templates
- Uncommited text colour to glass theme
- Supported browser indicator
- Configurable custom `>>>/${link}/` targets
- Optional global server-side anonymisation

###Changed
- Client rewritten mostly form scratch
- Meguca client compilation, install and upgrade procedures. See
README.md
- Pastebin embed height to 0.65 of viewport
- Client to single page application
- Faster builder.js client recompilation on change
- Upgraded dependancies, including minimal node.js version
- Google image search enabled by default
- Post menu only disappears on click either inside or outside the menu
- Render catalog server-side
- JSON API spec
- Metric ton of performance improvments
- Switched to 4chan-like thread expiry model with page limits and autosaging
- Persist `sage` in the email field
- Clicking on catalog images no longer opens new tab
- Criptographically secure mnemonics
- Always render inter-board navigation
- Updated JSON API post spec
- Restyle moderation selection checkbox
- Hide mnemonics from janitors
- Persistent JS script injections
- Don't preload all spoiler panes on postform render
- More efficient static asset caching

###Fixed
- Delay before r/a/dio banner appearing, when enabled
- Opus WebM parsing
- Random images generating small thumbnails
- Vagrant provisioning on OSX
- Disconnect favicon 404, if served from subdomain
- Navigating with inter-board post links
- Thread hiding
- Memory leak, if repetedly clicking on UTC clock
- Following a locked page bottom on post text input
- Ctrl clicking post links
- DOM bumping on reply shift on board pages
- (You) disapearing on page refresh
- Option exporting on Firefox
- Image hower previews not respecting aspect ration on Firefox
- Opening new tab when clicking audio controls in Firefox
- Dangling apostrophe in tripcode
- WebM expansion on Chrome for Android
- Overflow of admin notifications
- Purging hidden post list
- Outline around [Return] after clicking [Bottom]
- Deliberate server crashing, by sending certain websocket messages
- Incorrect process exit codes on termination

###Removed
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
- Imager deamon stub
- Dynamic post unloading (temporarily, until reimplementation)

##Breaking
- Dice storage in old threads
- Backlink generation in old threads
- Changed server entry point. Use `npm start` to start the server.
- Removed thread tagging. See `docs/migration.md`.

##0.11.3 - 2015-03-31
###Fixed
- Server crash, when thread expiry is not defined for all boards

##0.11.2 - 2015-03-19
###Fixed
- "Clear hidden" button now only renders on the General tab of options
- No word auto completion in the blockquote on mobile
- Auto quoting the header, when selected

##0.11.0 - 2015-03-15
###Added
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

###Fixed
- Administrator panel rendering on some configurations
- Silent desynchronisation on mobile
- Syncwatch imprecision
- Memory leak in syncwatch
- `Could not get response` error, when uploading with drag&drop
- Disapering replies on mobile

###Changed
- `npm install` now copies configuration files from examples on first
run
- Image hover previes now use the more responsive velocity.js library
- Moved r/a/dio API polling server-side. Now passed to the clients
through push notifications
- Patched upkeep/radio.js to use push notifications
- Made `#pyu` toggleable in the config.js
- Reduced the banner''s screen footprint
- Moved the separate archive deamon into a toggleable server module
- Log errors to meguca's root directory
- Updated init script sample
- Notification text colour to red for easier noticability
- `make clean` to also delete compiled JS and CSS files
- Vagrant now supplies all of the dependancies, including optional. Be
sure to run `vagrant provision` for existing VMs

###Removed
- Deployment script
- Default filter from the sample config.js
- Underline formatting for links
- Keyboard shortcuts options tab for mobile
- EXIF deletion script
- Autocompletion of text in the blockquote

##0.10.0 - 2015-02-24
###Added
- Vagrant support
- Readonly JSON API
- Store more values in post hashes (features that depend on these will
not work on threads created prior to 0.10.0)
- Thread catalog
- Option to disable staff IP tagging
- Adding custom URLs to board navigation
- youtu.be link embedding support

###Fixed
- io.js and node >= 0.12.0 compilation on OSX
- websocket hangups on node server disconnect
- Caching issues on Android

###Changed
- Expanded and reorganised documentation

##0.9.0 - 2015-02-17
###Added
- Chrome for Android home screen webapp support
- Option to unlock from page bottom on tab visibility loss
- Tabbed UI for options menu

###Fixed
- Administrator notifications on smaller screens
- ffmpeg RAM usage (still need plenty of RAM to thumbnail large WebM)

###Removed
- Anon hours

###Changed
- APNG detection and image duplicate comparison now use native C++ node
 addons
- Dynamic post unloading now off by default
- Syncwatch now always on
- All threads now have both [Expand] and [Last n] links
- Now using [io.js](https://iojs.org) instead of node.js. At the moment
of writing should still
  be compatible with node.js v0.12.x. When upgrading be sure to run
`make clean; npm update; npm install` from meguca's root directory

## 0.8.0 - 2015-01-30
### Changed
- Acquire most dependancies through npm
- Moved various settings from config.js to hot.js
- Duplicate image detection similar to findimagedupes.pl
- Replaced homebrew client builder with gulp.js (please run `# npm
install -g gulp` and rerun npm install)

### Fixed
- Syncwatch imprecision
- Report Recaptcha from HTTPS pages
- Illya dance + glass theme distortion on webkit browsers

### Removed
- Posted From siginitures

### Added
- Button for Fun Thread dispatching
- Client configuration push updates
- A fun in-model hook
- Clicking on desktop notifications focuses reply tab and post
- Displayed desktop notification memory
- Favicons indicate thread status
- Live administrator announcements
- Text spoiler keybind

## 0.7.0 - 2015-01-15
### Added
- Ban purge script
- mega.co.nz backup script
- Live-toggleable image-related settings
- Staff session purge script
- Expand all images button

### Removed
- Large thumbnail mode
- Japanese text-to-speach
- Composite spoilers
- WebM audio spoilers

### Changed
- Report emails activate moderation multiselection
- More complete post model extraction

## 0.6.0 - 2014-12-26
### Added
- [Top] link in all browsers
- Push websocket messages
- Online user counter

### Changed
- Mods can now ban

## 0.5.1 - 2014-12-22
### Added
- Hot-reloadable word filter
- Ability to render minimal page content (w/o scripts and CSS)
- Dynamic post unloading

### Removed
- Legacy wordfilter.js

### Changed
- Last N query string syntax
- Thread title sync to page title
- Disabled some broken capability on mobile browsers

### Fixed
- Faulty ban timeout assignment

## 0.4.0 - 2014-12-3
### Info
- Started versioning seperately from [doushio]
(https://github.com/lalcmellkmal/doushio)
