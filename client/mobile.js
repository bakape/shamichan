/*
 * Various mobile-only code
 */

let main = require('./main')

// TODO: Remove this module and related server-side logic/assets, once we
// build a separate mobile bundle

main.oneeSama.hook('spoilerTag', main.etc.touchable_spoiler_tag);
