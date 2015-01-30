# Change Log
All notable changes to this project will be documented in this file.
Project uses Semantic Versioning http://semver.org/

## 0.8.0 - 2015-01-30
### Changed
- Acquire most dependancies through npm
- Moved various settings from config.js to hot.js
- Duplicate image detection similar to findimagedupes.pl
- Replaced homebrew client builder with gulp.js (please run `# npm install -g gulp` and rerun npm install)

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
- Started versioning seperately from https://github.com/lalcmellkmal/doushio
